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

package asdf_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/ulikunitz/xz"
)

func TestExtractTarXz(t *testing.T) {
	t.Parallel()

	t.Run("extracts tar.xz archive", func(t *testing.T) {
		t.Parallel()

		tarXzTempDir := t.TempDir()

		archivePath := filepath.Join(tarXzTempDir, "test.tar.xz")
		CreateTestTarXz(t, archivePath, map[string]string{
			"test/file.txt": "file content",
		})

		destDir := filepath.Join(tarXzTempDir, "extracted-tarxz")
		require.NoError(t, asdf.ExtractTarXz(archivePath, destDir))

		content, err := os.ReadFile(filepath.Join(destDir, "test", "file.txt"))
		require.NoError(t, err)
		require.Equal(t, "file content", string(content))
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		err := asdf.ExtractTarXz("/nonexistent/archive.tar.xz", t.TempDir())
		require.Error(t, err)
	})

	t.Run("returns error for invalid xz file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.tar.xz")
		require.NoError(
			t,
			os.WriteFile(invalidPath, []byte("not an xz file"), asdf.CommonFilePermission),
		)

		err := asdf.ExtractTarXz(invalidPath, filepath.Join(tmpDir, "dest"))
		require.Error(t, err)
	})
}

func TestExtractTarGz(t *testing.T) {
	t.Parallel()

	t.Run("success scenarios", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			setup    func(*testing.T, string)
			validate func(*testing.T, string)
			name     string
		}{
			{
				name: "extracts tar.gz archive",
				setup: func(t *testing.T, path string) {
					t.Helper()
					CreateTestTarGz(t, path, map[string]string{
						"test/file.txt": "file content",
					})
				},
				validate: func(t *testing.T, dir string) {
					t.Helper()

					content, err := os.ReadFile(filepath.Join(dir, "test", "file.txt"))
					require.NoError(t, err)
					require.Equal(t, "file content", string(content))
				},
			},
			{
				name:  "extracts archive with directories",
				setup: CreateTestTarGzWithDirs,
				validate: func(t *testing.T, dir string) {
					t.Helper()

					info, err := os.Stat(filepath.Join(dir, "mydir"))
					require.NoError(t, err)
					require.True(t, info.IsDir())
				},
			},
			{
				name:  "extracts archive with symlinks",
				setup: CreateTestTarGzWithSymlink,
				validate: func(t *testing.T, dir string) {
					t.Helper()

					linkTarget, err := os.Readlink(filepath.Join(dir, "link"))
					require.NoError(t, err)
					require.Equal(t, "target.txt", linkTarget)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tempDir := t.TempDir()
				archivePath := filepath.Join(tempDir, "archive.tar.gz")
				tt.setup(t, archivePath)

				destDir := filepath.Join(tempDir, "extracted")
				require.NoError(t, asdf.ExtractTarGz(archivePath, destDir))
				tt.validate(t, destDir)
			})
		}
	})

	t.Run("error scenarios", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			setup     func(*testing.T) string
			name      string
			errSubstr string
			wantErr   bool
		}{
			{
				name:    "nonexistent file",
				setup:   func(*testing.T) string { return "/nonexistent/archive.tar.gz" },
				wantErr: true,
			},
			{
				name: "invalid gzip file",
				setup: func(t *testing.T) string {
					t.Helper()

					path := filepath.Join(t.TempDir(), "invalid.tar.gz")
					require.NoError(
						t,
						os.WriteFile(path, []byte("not a gzip file"), asdf.CommonFilePermission),
					)

					return path
				},
				wantErr: true,
			},
			{
				name: "directory traversal",
				setup: func(t *testing.T) string {
					t.Helper()

					path := filepath.Join(t.TempDir(), "traversal.tar.gz")
					CreateTestTarGzWithTraversal(t, path)

					return path
				},
				wantErr:   true,
				errSubstr: "invalid file path",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				archivePath := tt.setup(t)

				err := asdf.ExtractTarGz(archivePath, t.TempDir())
				if tt.wantErr {
					require.Error(t, err)

					if tt.errSubstr != "" {
						require.Contains(t, err.Error(), tt.errSubstr)
					}
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestExtractZip(t *testing.T) {
	t.Parallel()

	t.Run("success scenarios", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			setup    func(*testing.T, string)
			validate func(*testing.T, string)
			name     string
		}{
			{
				name: "extracts zip archive",
				setup: func(t *testing.T, path string) {
					t.Helper()
					CreateTestZip(t, path, map[string]string{
						"test/file.txt": "file content",
					})
				},
				validate: func(t *testing.T, dir string) {
					t.Helper()

					content, err := os.ReadFile(filepath.Join(dir, "test", "file.txt"))
					require.NoError(t, err)
					require.Equal(t, "file content", string(content))
				},
			},
			{
				name:  "extracts zip with directories",
				setup: CreateTestZipWithDirs,
				validate: func(t *testing.T, dir string) {
					t.Helper()

					info, err := os.Stat(filepath.Join(dir, "mydir"))
					require.NoError(t, err)
					require.True(t, info.IsDir())

					content, err := os.ReadFile(filepath.Join(dir, "mydir", "file.txt"))
					require.NoError(t, err)
					require.Equal(t, "file in dir", string(content))
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tempDir := t.TempDir()
				archivePath := filepath.Join(tempDir, "archive.zip")
				tt.setup(t, archivePath)

				destDir := filepath.Join(tempDir, "extracted")
				require.NoError(t, asdf.ExtractZip(archivePath, destDir))
				tt.validate(t, destDir)
			})
		}
	})

	t.Run("error scenarios", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			setup     func(*testing.T) string
			name      string
			errSubstr string
			wantErr   bool
		}{
			{
				name:    "nonexistent file",
				setup:   func(*testing.T) string { return "/nonexistent/archive.zip" },
				wantErr: true,
			},
			{
				name: "directory traversal",
				setup: func(t *testing.T) string {
					t.Helper()

					path := filepath.Join(t.TempDir(), "traversal.zip")
					CreateTestZipWithTraversal(t, path)

					return path
				},
				wantErr:   true,
				errSubstr: "invalid file path",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				archivePath := tt.setup(t)

				err := asdf.ExtractZip(archivePath, t.TempDir())
				if tt.wantErr {
					require.Error(t, err)

					if tt.errSubstr != "" {
						require.Contains(t, err.Error(), tt.errSubstr)
					}
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestExtractGz(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup     func(*testing.T) string
		validate  func(*testing.T, string)
		name      string
		errSubstr string
		wantErr   bool
	}{
		{
			name: "extracts gz file",
			setup: func(t *testing.T) string {
				t.Helper()

				path := filepath.Join(t.TempDir(), "test.gz")
				CreateTestGz(t, path, "file content")

				return path
			},
			validate: func(t *testing.T, path string) {
				t.Helper() // gz file test

				content, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, "file content", string(content))
			},
		},
		{
			name: "invalid gz file",
			setup: func(t *testing.T) string {
				t.Helper()

				path := filepath.Join(t.TempDir(), "invalid.gz")
				require.NoError(
					t,
					os.WriteFile(path, []byte("not a gz file"), asdf.CommonFilePermission),
				)

				return path
			},
			wantErr: true,
		},
		{
			name:    "nonexistent file",
			setup:   func(*testing.T) string { return "/nonexistent/file.gz" },
			wantErr: true,
		},
		{
			name: "empty file",
			setup: func(*testing.T) string {
				path := filepath.Join(t.TempDir(), "empty.gz")
				require.NoError(t, os.WriteFile(path, nil, asdf.CommonFilePermission))

				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			archivePath := tt.setup(t)
			destPath := filepath.Join(t.TempDir(), "out")

			err := asdf.ExtractGz(archivePath, destPath)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errSubstr != "" {
					require.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)

				if tt.validate != nil {
					tt.validate(t, destPath)
				}
			}
		})
	}
}

func TestIsPathWithinDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     func(base string) string
		name     string
		expected bool
	}{
		{
			name:     "inside directory",
			path:     func(base string) string { return filepath.Join(base, "sub", "file.txt") },
			expected: true,
		},
		{
			name:     "outside directory",
			path:     func(base string) string { return filepath.Join(filepath.Dir(base), "other", "file.txt") },
			expected: false,
		},
		{
			name:     "same directory",
			path:     func(base string) string { return base },
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := filepath.Join(t.TempDir(), "base")
			require.Equal(t, tt.expected, asdf.IsPathWithinDirForTests(tt.path(base), base))
		})
	}
}

func TestLimitedArchiveWriter(t *testing.T) {
	t.Parallel()

	t.Run("enforces limits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			errSubstr string
			write     []string
			maxTotal  int64
			maxFile   int64
			wantTotal int64
			wantErr   bool
		}{
			{
				name:      "within limits",
				maxTotal:  10,
				maxFile:   5,
				write:     []string{"hello"},
				wantTotal: 5,
			},
			{
				name:      "exceeds file limit",
				maxTotal:  20,
				maxFile:   4,
				write:     []string{"hello"},
				wantErr:   true,
				wantTotal: 4,
			},
			{
				name:      "exceeds total limit",
				maxTotal:  8,
				maxFile:   10,
				write:     []string{"hello", "world"},
				wantErr:   true,
				wantTotal: 8,
			},
			{
				name:      "zero limits",
				maxTotal:  0,
				maxFile:   0,
				write:     []string{"test"},
				wantErr:   true,
				errSubstr: "invalid archive size limits",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var (
					buf   bytes.Buffer
					total int64
				)

				writer := asdf.NewLimitedArchiveWriterForTests(
					&buf,
					&total,
					tt.maxTotal,
					tt.maxFile,
				)

				var err error
				for _, s := range tt.write {
					_, err = writer.Write([]byte(s))
					if err != nil {
						break
					}
				}

				if tt.wantErr {
					require.Error(t, err)

					if tt.errSubstr != "" {
						require.Contains(t, err.Error(), tt.errSubstr)
					}
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.wantTotal, total)
				}
			})
		}
	})

	t.Run("nil total counter", func(t *testing.T) {
		t.Parallel()

		writer := asdf.NewLimitedArchiveWriterForTests(&bytes.Buffer{}, nil, 100, 50)
		_, err := writer.Write([]byte("test"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "total counter is nil")
	})

	t.Run("accumulates total across instances", func(t *testing.T) {
		t.Parallel()

		var (
			total int64
			buf   bytes.Buffer
		)

		w1 := asdf.NewLimitedArchiveWriterForTests(&buf, &total, 10, 10)
		_, err := w1.Write([]byte("hello"))
		require.NoError(t, err)
		require.Equal(t, int64(5), total)

		w2 := asdf.NewLimitedArchiveWriterForTests(&buf, &total, 10, 10)

		_, err = w2.Write([]byte("world!"))
		require.Error(t, err)
		require.Equal(t, int64(10), total)
	})
}

// Helpers

func CreateTestTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	createArchive(t, path, func(tw *tar.Writer) {
		for name, content := range files {
			require.NoError(t, tw.WriteHeader(&tar.Header{
				Name: name,
				Mode: int64(asdf.TarFilePermission),
				Size: int64(len(content)),
			}))

			_, err := tw.Write([]byte(content))
			require.NoError(t, err)
		}
	})
}

func CreateTestTarGzWithDirs(t *testing.T, path string) {
	t.Helper()
	createArchive(t, path, func(tw *tar.Writer) {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     "mydir/",
			Mode:     int64(asdf.CommonDirectoryPermission),
			Typeflag: tar.TypeDir,
		}))

		content := "file in dir"
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: "mydir/file.txt",
			Mode: int64(asdf.TarFilePermission),
			Size: int64(len(content)),
		}))

		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	})
}

func CreateTestTarGzWithSymlink(t *testing.T, path string) {
	t.Helper()
	createArchive(t, path, func(tw *tar.Writer) {
		content := "target content"
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: "target.txt",
			Mode: int64(asdf.TarFilePermission),
			Size: int64(len(content)),
		}))

		_, err := tw.Write([]byte(content))
		require.NoError(t, err)

		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     "link",
			Mode:     int64(asdf.TarLinkPermission),
			Typeflag: tar.TypeSymlink,
			Linkname: "target.txt",
		}))
	})
}

func CreateTestTarGzWithTraversal(t *testing.T, path string) {
	t.Helper()
	createArchive(t, path, func(tw *tar.Writer) {
		content := "malicious content"
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: "../../../etc/malicious.txt",
			Mode: int64(asdf.TarFilePermission),
			Size: int64(len(content)),
		}))

		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	})
}

func createArchive(t *testing.T, path string, writerFunc func(*tar.Writer)) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	writerFunc(tw)
}

func CreateTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)

	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	for name, content := range files {
		f, err := zipw.Create(name)
		require.NoError(t, err)

		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
}

func CreateTestZipWithTraversal(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)

	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	f, err := zipw.Create("../../../etc/malicious.txt")
	require.NoError(t, err)

	_, err = f.Write([]byte("malicious"))
	require.NoError(t, err)
}

func CreateTestZipWithDirs(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)

	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	header := &zip.FileHeader{Name: "mydir/", Method: zip.Store}
	header.SetMode(asdf.CommonDirectoryPermission | os.ModeDir)

	_, err = zipw.CreateHeader(header)
	require.NoError(t, err)

	fileHeader := &zip.FileHeader{Name: "mydir/file.txt", Method: zip.Store}
	fileHeader.SetMode(asdf.TarFilePermission)

	f, err := zipw.CreateHeader(fileHeader)
	require.NoError(t, err)

	_, err = f.Write([]byte("file in dir"))
	require.NoError(t, err)
}

func CreateTestGz(t *testing.T, path, content string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)

	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	_, err = gzw.Write([]byte(content))
	require.NoError(t, err)
}

func CreateTestTarXz(t *testing.T, path string, files map[string]string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)

	defer file.Close()

	xzw, err := xz.NewWriter(file)
	require.NoError(t, err)

	defer xzw.Close()

	tw := tar.NewWriter(xzw)
	defer tw.Close()

	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: int64(asdf.TarFilePermission),
			Size: int64(len(content)),
		}))

		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
}
