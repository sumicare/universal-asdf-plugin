//
// Copyright (c) 2025 Sumicare
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asdf

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

const (
	// TarFilePermission is the file mode used for non-executable entries in the test Go tar archives.
	TarFilePermission os.FileMode = 0o644
	// TarLinkPermission is the file mode used for link entries in the test Go tar archives.
	TarLinkPermission os.FileMode = 0o777
	// maxArchiveBytes is the maximum total number of bytes that can be written across all extracted archive entries.
	maxArchiveBytes int64 = 1 << 30
	// maxArchiveFileBytes is the maximum size in bytes permitted for a single extracted archive entry.
	maxArchiveFileBytes int64 = 512 << 20
)

var (
	// errTarEntryTooLarge indicates a single tar entry exceeds the allowed maximum size.
	errTarEntryTooLarge = errors.New("tar entry too large")
	// errZipEntryTooLarge indicates a single zip entry exceeds the allowed maximum size.
	errZipEntryTooLarge = errors.New("zip entry too large")
	// errLimitedArchiveWriterTotalIsNil indicates the shared total counter was not provided.
	errLimitedArchiveWriterTotalIsNil = errors.New("limitedArchiveWriter total counter is nil")
	// errInvalidArchiveSizeLimits indicates the configured archive size limits are invalid.
	errInvalidArchiveSizeLimits = errors.New("invalid archive size limits")
	// errArchiveSizeLimitExceeded indicates an archive exceeded one of the configured size limits.
	errArchiveSizeLimitExceeded = errors.New("archive size limit exceeded")
)

// extractTarEntries extracts all entries from a tar reader to the destination directory.
func extractTarEntries(tr *tar.Reader, destDir string) error {
	var totalWritten int64

	cleanDestDir := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		target := filepath.Join(cleanDestDir, filepath.Clean(header.Name))
		if !isPathWithinDir(target, cleanDestDir) {
			return fmt.Errorf("%w: %s", errInvalidArchiveFilePathTar, header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, header.FileInfo().Mode().Perm()); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}

		case tar.TypeReg:
			if header.Size > maxArchiveFileBytes {
				return fmt.Errorf("%w: %d bytes", errTarEntryTooLarge, header.Size)
			}

			if err := os.MkdirAll(filepath.Dir(target), CommonDirectoryPermission); err != nil {
				return fmt.Errorf("creating parent directory: %w", err)
			}

			fileMode := header.FileInfo().Mode()

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fileMode)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}

			lw := &limitedArchiveWriter{
				w:        outFile,
				total:    &totalWritten,
				maxTotal: maxArchiveBytes,
				maxFile:  maxArchiveFileBytes,
			}

			if _, err := io.Copy(lw, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}

			outFile.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), CommonDirectoryPermission); err != nil {
				return fmt.Errorf("creating parent directory: %w", err)
			}

			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("creating symlink %s: %w", target, err)
			}
		}
	}

	return nil
}

// ExtractTarGz extracts a .tar.gz file to the destination directory.
func ExtractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	return extractTarEntries(tar.NewReader(gzr), destDir)
}

// ExtractTarXz extracts a .tar.xz file to the destination directory.
func ExtractTarXz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating xz reader: %w", err)
	}

	return extractTarEntries(tar.NewReader(xzr), destDir)
}

// ExtractZip extracts a .zip file to the destination directory.
func ExtractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("opening zip archive: %w", err)
	}
	defer reader.Close()

	var totalWritten int64

	cleanDestDir := filepath.Clean(destDir)

	for _, zipFile := range reader.File {
		target := filepath.Join(cleanDestDir, filepath.Clean(zipFile.Name))
		if !isPathWithinDir(target, cleanDestDir) {
			return fmt.Errorf("%w: %s", errInvalidArchiveFilePathZip, zipFile.Name)
		}

		if zipFile.UncompressedSize64 > uint64(maxArchiveFileBytes) {
			return fmt.Errorf("%w: %d bytes", errZipEntryTooLarge, zipFile.UncompressedSize64)
		}

		if zipFile.FileInfo().IsDir() {
			if err := os.MkdirAll(target, CommonDirectoryPermission); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}

			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), CommonDirectoryPermission); err != nil {
			return fmt.Errorf("creating parent directory: %w", err)
		}

		rc, err := zipFile.Open()
		if err != nil {
			return fmt.Errorf("opening file in archive: %w", err)
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, zipFile.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("creating file %s: %w", target, err)
		}

		lw := &limitedArchiveWriter{
			w:        outFile,
			total:    &totalWritten,
			maxTotal: maxArchiveBytes,
			maxFile:  maxArchiveFileBytes,
		}

		if _, err := io.Copy(lw, rc); err != nil { //nolint:gosec // G110: decompressed size is bounded by limitedArchiveWriter
			outFile.Close()
			rc.Close()

			return fmt.Errorf("writing file %s: %w", target, err)
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}

// ExtractGz extracts a .gz file to the destination path.
func ExtractGz(gzPath, destPath string) error {
	gzFile, err := os.Open(gzPath)
	if err != nil {
		return fmt.Errorf("opening gz file: %w", err)
	}
	defer gzFile.Close()

	gzr, err := gzip.NewReader(gzFile)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, CommonFilePermission)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	var totalWritten int64

	lw := &limitedArchiveWriter{
		w:        outFile,
		total:    &totalWritten,
		maxTotal: maxArchiveBytes,
		maxFile:  maxArchiveFileBytes,
	}

	if _, err := io.Copy(lw, gzr); err != nil { //nolint:gosec // G110: decompressed size is bounded by limitedArchiveWriter
		return fmt.Errorf("extracting gz: %w", err)
	}

	return nil
}

// limitedArchiveWriter is a writer that limits the total size of the archive.
type limitedArchiveWriter struct {
	w        io.Writer
	total    *int64
	maxTotal int64
	maxFile  int64
	written  int64
}

// Write implements io.Writer.
func (writer *limitedArchiveWriter) Write(buff []byte) (int, error) {
	if writer.total == nil {
		return 0, errLimitedArchiveWriterTotalIsNil
	}

	if writer.maxFile <= 0 || writer.maxTotal <= 0 {
		return 0, errInvalidArchiveSizeLimits
	}

	remainingFile := writer.maxFile - writer.written

	remainingTotal := writer.maxTotal - *writer.total
	if remainingFile <= 0 || remainingTotal <= 0 {
		return 0, errArchiveSizeLimitExceeded
	}

	toWrite := min(min(int64(len(buff)), remainingFile), remainingTotal)

	if toWrite <= 0 {
		return 0, errArchiveSizeLimitExceeded
	}

	numBytes, err := writer.w.Write(buff[:toWrite])

	writer.written += int64(numBytes)
	*writer.total += int64(numBytes)

	if err != nil {
		return numBytes, err
	}

	if int64(numBytes) < int64(len(buff)) {
		return numBytes, errArchiveSizeLimitExceeded
	}

	return numBytes, nil
}

// isPathWithinDir checks if the path is within the directory.
func isPathWithinDir(path, dir string) bool {
	cleanDir := filepath.Clean(dir)
	cleanPath := filepath.Clean(path)

	if cleanDir == cleanPath {
		return true
	}

	return strings.HasPrefix(cleanPath, cleanDir+string(os.PathSeparator))
}
