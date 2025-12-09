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
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"

	"github.com/ulikunitz/xz"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Archive helpers", func() {
	Describe("ExtractTarGz", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "extract-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("ExtractTarXz", func() {
			var tarXzTempDir string

			BeforeEach(func() {
				var err error
				tarXzTempDir, err = os.MkdirTemp("", "extract-tarxz-test-*")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.RemoveAll(tarXzTempDir)
			})

			It("extracts tar.xz archive", func() {
				archivePath := filepath.Join(tarXzTempDir, "test.tar.xz")
				CreateTestTarXz(archivePath, map[string]string{
					"test/file.txt": "file content",
				})

				destDir := filepath.Join(tarXzTempDir, "extracted-tarxz")
				err := ExtractTarXz(archivePath, destDir)
				Expect(err).NotTo(HaveOccurred())

				content, err := os.ReadFile(filepath.Join(destDir, "test", "file.txt"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("file content"))
			})
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("extracts tar.gz archive", func() {
			archivePath := filepath.Join(tempDir, "test.tar.gz")
			CreateTestTarGz(archivePath, map[string]string{
				"test/file.txt": "file content",
			})

			destDir := filepath.Join(tempDir, "extracted")
			err := ExtractTarGz(archivePath, destDir)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(filepath.Join(destDir, "test", "file.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("file content"))
		})

		It("extracts archive with directories", func() {
			archivePath := filepath.Join(tempDir, "test-dirs.tar.gz")
			CreateTestTarGzWithDirs(archivePath)

			destDir := filepath.Join(tempDir, "extracted-dirs")
			err := ExtractTarGz(archivePath, destDir)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(filepath.Join(destDir, "mydir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})

		It("extracts archive with symlinks", func() {
			archivePath := filepath.Join(tempDir, "test-symlink.tar.gz")
			CreateTestTarGzWithSymlink(archivePath)

			destDir := filepath.Join(tempDir, "extracted-symlink")
			err := ExtractTarGz(archivePath, destDir)
			Expect(err).NotTo(HaveOccurred())

			linkTarget, err := os.Readlink(filepath.Join(destDir, "link"))
			Expect(err).NotTo(HaveOccurred())
			Expect(linkTarget).To(Equal("target.txt"))
		})

		It("returns error for nonexistent file", func() {
			err := ExtractTarGz("/nonexistent/archive.tar.gz", tempDir)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid gzip file", func() {
			invalidPath := filepath.Join(tempDir, "invalid.tar.gz")
			err := os.WriteFile(invalidPath, []byte("not a gzip file"), CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = ExtractTarGz(invalidPath, filepath.Join(tempDir, "out"))
			Expect(err).To(HaveOccurred())
		})

		It("returns error for directory traversal attempt", func() {
			archivePath := filepath.Join(tempDir, "traversal.tar.gz")
			CreateTestTarGzWithTraversal(archivePath)

			destDir := filepath.Join(tempDir, "extracted-traversal")
			err := ExtractTarGz(archivePath, destDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid file path"))
		})
	})

	Describe("ExtractZip", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "extract-zip-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("extracts zip archive", func() {
			archivePath := filepath.Join(tempDir, "test.zip")
			CreateTestZip(archivePath, map[string]string{
				"test/file.txt": "file content",
			})

			destDir := filepath.Join(tempDir, "extracted")
			err := ExtractZip(archivePath, destDir)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(filepath.Join(destDir, "test", "file.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("file content"))
		})

		It("returns error for nonexistent file", func() {
			err := ExtractZip("/nonexistent/archive.zip", tempDir)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for directory traversal", func() {
			archivePath := filepath.Join(tempDir, "traversal.zip")
			CreateTestZipWithTraversal(archivePath)

			destDir := filepath.Join(tempDir, "extracted-traversal")
			err := ExtractZip(archivePath, destDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid file path"))
		})

		It("extracts zip with directories", func() {
			archivePath := filepath.Join(tempDir, "dirs.zip")
			CreateTestZipWithDirs(archivePath)

			destDir := filepath.Join(tempDir, "extracted-dirs")
			err := ExtractZip(archivePath, destDir)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(filepath.Join(destDir, "mydir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())

			content, err := os.ReadFile(filepath.Join(destDir, "mydir", "file.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("file in dir"))
		})
	})

	Describe("ExtractGz", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "extract-gz-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("extracts gz file", func() {
			archivePath := filepath.Join(tempDir, "test.gz")
			CreateTestGz(archivePath, "file content")

			destPath := filepath.Join(tempDir, "extracted-file")
			err := ExtractGz(archivePath, destPath)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("file content"))
		})

		It("returns error for invalid gz file", func() {
			invalidPath := filepath.Join(tempDir, "invalid.gz")
			err := os.WriteFile(invalidPath, []byte("not a gz file"), CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = ExtractGz(invalidPath, filepath.Join(tempDir, "out"))
			Expect(err).To(HaveOccurred())
		})

		It("returns error for nonexistent file", func() {
			err := ExtractGz("/nonexistent/file.gz", filepath.Join(tempDir, "out"))
			Expect(err).To(HaveOccurred())
		})

		It("returns error for empty file", func() {
			emptyPath := filepath.Join(tempDir, "empty.gz")
			err := os.WriteFile(emptyPath, nil, CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = ExtractGz(emptyPath, filepath.Join(tempDir, "out"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("isPathWithinDir", func() {
		It("returns true for paths inside directory", func() {
			base := filepath.Join(os.TempDir(), "base")
			path := filepath.Join(base, "sub", "file.txt")
			Expect(isPathWithinDir(path, base)).To(BeTrue())
		})

		It("returns false for paths outside directory", func() {
			base := filepath.Join(os.TempDir(), "base")
			path := filepath.Join(os.TempDir(), "other", "file.txt")
			Expect(isPathWithinDir(path, base)).To(BeFalse())
		})

		It("handles equal paths", func() {
			base := filepath.Join(os.TempDir(), "base")
			Expect(isPathWithinDir(base, base)).To(BeTrue())
		})
	})

	Describe("limitedArchiveWriter", func() {
		It("enforces per-file and total limits", func() {
			var buf bytes.Buffer
			var total int64

			writer := &limitedArchiveWriter{
				w:        &buf,
				total:    &total,
				maxTotal: 8,
				maxFile:  5,
			}

			n, err := writer.Write([]byte("hello"))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(total).To(Equal(int64(5)))
			Expect(buf.String()).To(Equal("hello"))

			_, err = writer.Write([]byte("world"))
			Expect(err).To(HaveOccurred())
		})

		It("returns error when total counter is nil", func() {
			var buf bytes.Buffer

			writer := &limitedArchiveWriter{
				w:        &buf,
				total:    nil,
				maxTotal: 100,
				maxFile:  50,
			}

			_, err := writer.Write([]byte("test"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("total counter is nil"))
		})

		It("returns error when limits are zero or negative", func() {
			var buf bytes.Buffer
			var total int64

			writer := &limitedArchiveWriter{
				w:        &buf,
				total:    &total,
				maxTotal: 0,
				maxFile:  0,
			}

			_, err := writer.Write([]byte("test"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid archive size limits"))
		})

		It("enforces total limit across multiple files", func() {
			var buf bytes.Buffer
			var total int64

			writer := &limitedArchiveWriter{
				w:        &buf,
				total:    &total,
				maxTotal: 10,
				maxFile:  100,
			}

			numBytes, err := writer.Write([]byte("hello!"))
			Expect(err).NotTo(HaveOccurred())
			Expect(numBytes).To(Equal(6))
			Expect(total).To(Equal(int64(6)))

			writer2 := &limitedArchiveWriter{
				w:        &buf,
				total:    &total,
				maxTotal: 10,
				maxFile:  100,
			}

			numBytes, err = writer2.Write([]byte("world!"))
			Expect(err).To(HaveOccurred())
			Expect(numBytes).To(Equal(4))
			Expect(total).To(Equal(int64(10)))
		})
	})
})

// CreateTestTarGz creates a tar.gz archive at the given path with the
// provided files. It mirrors the layout used by real plugin downloads
// so that ExtractTarGz can be exercised in a controlled way.
func CreateTestTarGz(path string, files map[string]string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: int64(TarFilePermission),
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(header)
		Expect(err).NotTo(HaveOccurred())

		_, err = tw.Write([]byte(content))
		Expect(err).NotTo(HaveOccurred())
	}
}

// CreateTestTarGzWithDirs creates a tar.gz archive containing
// directories and files. It is used to verify directory handling during
// extraction.
func CreateTestTarGzWithDirs(path string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err = tw.WriteHeader(&tar.Header{
		Name:     "mydir/",
		Mode:     int64(CommonDirectoryPermission),
		Typeflag: tar.TypeDir,
	})
	Expect(err).NotTo(HaveOccurred())

	content := "file in dir"

	err = tw.WriteHeader(&tar.Header{
		Name: "mydir/file.txt",
		Mode: int64(TarFilePermission),
		Size: int64(len(content)),
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = tw.Write([]byte(content))
	Expect(err).NotTo(HaveOccurred())
}

// CreateTestTarGzWithSymlink creates a tar.gz archive containing a file
// and a symlink. It is used to test symlink handling during extraction
// without depending on a real release archive.
func CreateTestTarGzWithSymlink(path string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	content := "target content"

	err = tw.WriteHeader(&tar.Header{
		Name: "target.txt",
		Mode: int64(TarFilePermission),
		Size: int64(len(content)),
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = tw.Write([]byte(content))
	Expect(err).NotTo(HaveOccurred())

	err = tw.WriteHeader(&tar.Header{
		Name:     "link",
		Mode:     int64(TarLinkPermission),
		Typeflag: tar.TypeSymlink,
		Linkname: "target.txt",
	})
	Expect(err).NotTo(HaveOccurred())
}

// CreateTestTarGzWithTraversal creates a tar.gz archive with a path
// traversal entry. It is used to ensure ExtractTarGz protects against
// directory traversal.
func CreateTestTarGzWithTraversal(path string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	content := "malicious content"

	err = tw.WriteHeader(&tar.Header{
		Name: "../../../etc/malicious.txt",
		Mode: int64(TarFilePermission),
		Size: int64(len(content)),
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = tw.Write([]byte(content))
	Expect(err).NotTo(HaveOccurred())
}

// CreateTestZip creates a zip archive at the given path with the
// provided files. The resulting archive is intentionally small so
// ExtractZip tests stay fast.
func CreateTestZip(path string, files map[string]string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	for name, content := range files {
		f, err := zipw.Create(name)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.Write([]byte(content))
		Expect(err).NotTo(HaveOccurred())
	}
}

// CreateTestZipWithTraversal creates a zip archive containing a path
// traversal entry. It is used to ensure ExtractZip protects against
// directory traversal.
func CreateTestZipWithTraversal(path string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	f, err := zipw.Create("../../../etc/malicious.txt")
	Expect(err).NotTo(HaveOccurred())

	_, err = f.Write([]byte("malicious"))
	Expect(err).NotTo(HaveOccurred())
}

// CreateTestZipWithDirs creates a zip archive containing directories and
// files so that zip extraction can be validated against nested
// structures.
func CreateTestZipWithDirs(path string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	header := &zip.FileHeader{
		Name:   "mydir/",
		Method: zip.Store,
	}
	header.SetMode(CommonDirectoryPermission | os.ModeDir)

	_, err = zipw.CreateHeader(header)
	Expect(err).NotTo(HaveOccurred())

	fileHeader := &zip.FileHeader{
		Name:   "mydir/file.txt",
		Method: zip.Store,
	}
	fileHeader.SetMode(TarFilePermission)

	f, err := zipw.CreateHeader(fileHeader)
	Expect(err).NotTo(HaveOccurred())

	_, err = f.Write([]byte("file in dir"))
	Expect(err).NotTo(HaveOccurred())
}

// CreateTestGz creates a gzip archive with the given content and is
// used to test ExtractGz independently from tar or zip handling.
func CreateTestGz(path, content string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	_, err = gzw.Write([]byte(content))
	Expect(err).NotTo(HaveOccurred())
}

// CreateTestTarXz creates a tar.xz archive at the given path with the
// provided files. It mirrors the layout used by real plugin downloads
// so that ExtractTarXz can be exercised in a controlled way.
func CreateTestTarXz(path string, files map[string]string) {
	file, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	xzw, err := xz.NewWriter(file)
	Expect(err).NotTo(HaveOccurred())

	defer xzw.Close()

	tw := tar.NewWriter(xzw)
	defer tw.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: int64(TarFilePermission),
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(header)
		Expect(err).NotTo(HaveOccurred())

		_, err = tw.Write([]byte(content))
		Expect(err).NotTo(HaveOccurred())
	}
}
