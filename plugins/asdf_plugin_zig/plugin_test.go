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

package asdf_plugin_zig

import (
	"archive/tar"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var _ = Describe("Zig Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("zig"))
		})
	})

	Describe("fetchIndex error cases", func() {
		It("returns error on non-200 response", func() {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/index.json" {
					writer.WriteHeader(http.StatusInternalServerError)
					return
				}

				writer.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			_, err := plugin.fetchIndex(context.Background())
			Expect(err).To(HaveOccurred())
		})

		It("returns error on invalid JSON response", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/index.json" {
					_, _ = w.Write([]byte("{")) //nolint:errcheck // test handler
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			_, err := plugin.fetchIndex(context.Background())
			Expect(err).To(HaveOccurred())
		})

		It("skips non-release entries when decoding index", func() {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/index.json" {

					indexJSON := `{"0.14.0": {"date": {"foo": "bar"}, "x86_64-linux": {"tarball": "http://example.com/zig.tar.xz", "shasum": "", "size": "0"}}}`
					_, _ = writer.Write([]byte(indexJSON)) //nolint:errcheck // test handler

					return
				}

				writer.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			index, err := plugin.fetchIndex(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(index).To(HaveKey("0.14.0"))
			Expect(index["0.14.0"]).To(HaveKey("x86_64-linux"))
			Expect(index["0.14.0"]).NotTo(HaveKey("date"))
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := New()
			Expect(plugin.Name()).To(Equal("zig"))
		})
	})

	Describe("ListBinPaths", func() {
		It("returns bin paths", func() {
			plugin := New()
			paths := plugin.ListBinPaths()
			Expect(paths).To(Equal("."))
		})
	})

	Describe("ExecEnv", func() {
		It("returns empty environment", func() {
			plugin := New()
			env := plugin.ExecEnv("/tmp/install")
			Expect(env).To(BeEmpty())
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns empty list", func() {
			plugin := New()
			files := plugin.ListLegacyFilenames()
			Expect(files).To(BeEmpty())
		})
	})

	Describe("ParseLegacyFile", func() {
		It("parses version from file", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			versionFile := filepath.Join(tempDir, ".zig-version")
			err = os.WriteFile(versionFile, []byte("0.14.0\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := plugin.ParseLegacyFile(versionFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("0.14.0\n"))
		})

		It("returns error when file does not exist", func() {
			plugin := New()
			_, err := plugin.ParseLegacyFile("/nonexistent/zig-version")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-uninstall-*")
			Expect(err).NotTo(HaveOccurred())

			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			_, err = os.Stat(installPath)
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Uninstall(context.Background(), installPath)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(installPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("Help", func() {
		It("returns help information", func() {
			plugin := New()
			help := plugin.Help()
			Expect(help.Overview).To(ContainSubstring("Zig"))
			Expect(help.Deps).NotTo(BeEmpty())
			Expect(help.Config).NotTo(BeEmpty())
			Expect(help.Links).To(ContainSubstring("ziglang.org"))
		})
	})

	Describe("ListAll", func() {
		It("lists Zig versions from mock index", func() {
			server := newZigIndexServer(map[string]map[string]ZigRelease{
				"0.13.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
				"0.14.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
			})
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElement("0.13.0"))
			Expect(versions).To(ContainElement("0.14.0"))
		})
	})

	Describe("LatestStable", func() {
		It("returns latest stable version from mock index", func() {
			server := newZigIndexServer(map[string]map[string]ZigRelease{
				"0.13.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
				"0.14.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
			})
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("0.14.0"))
		})

		It("filters by pattern from mock index", func() {
			server := newZigIndexServer(map[string]map[string]ZigRelease{
				"0.13.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
				"0.13.1": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
				"0.14.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
			})
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "0.13")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("0.13.1"))
		})

		It("returns error when no versions match pattern in mock index", func() {
			server := newZigIndexServer(map[string]map[string]ZigRelease{
				"0.13.0": {"x86_64-linux": {Tarball: "ignored", Shasum: "", Size: "0"}},
			})
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			_, err := plugin.LatestStable(context.Background(), "99.99")
			Expect(err).To(HaveOccurred())
		})

		It("returns error when no versions are available", func() {
			server := newZigIndexServer(make(map[string]map[string]ZigRelease))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			_, err := plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download", func() {
		It("uses cached tarball when already downloaded", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			archivePath := filepath.Join(tempDir, "zig.tar.xz")

			content := make([]byte, 0, 2048)
			for i := range content {
				content = append(content, byte(i%256))
			}
			Expect(os.WriteFile(archivePath, content, asdf.CommonFilePermission)).To(Succeed())

			Expect(plugin.Download(context.Background(), "0.14.0", tempDir)).To(Succeed())
		})

		It("downloads tarball from mock index", func() {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				switch req.URL.Path {
				case "/index.json":
					indexJSON := fmt.Sprintf(`{"0.14.0": {"x86_64-linux": {"tarball": "http://%s/zig-0.14.0.tar.xz", "shasum": "", "size": "0"}}}`, req.Host)
					_, _ = writer.Write([]byte(indexJSON)) //nolint:errcheck // test handler
				case "/zig-0.14.0.tar.xz":
					_, _ = writer.Write([]byte("fake-binary-data")) //nolint:errcheck // test handler
				default:
					writer.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			Expect(plugin.Download(context.Background(), "0.14.0", tempDir)).To(Succeed())

			archivePath := filepath.Join(tempDir, "zig.tar.xz")
			info, err := os.Stat(archivePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Size()).To(BeNumerically(">", 0))
		})

		It("returns error when version not found", func() {
			server := newZigIndexServer(make(map[string]map[string]ZigRelease))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "0.14.0", tempDir)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when no release exists for current platform", func() {
			server := newZigIndexServer(map[string]map[string]ZigRelease{
				"0.14.0": {"some-other-platform": {Tarball: "ignored", Shasum: "", Size: "0"}},
			})
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-download-noplatform-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "0.14.0", tempDir)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when download of tarball fails", func() {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				switch req.URL.Path {
				case "/index.json":
					indexJSON := fmt.Sprintf(`{"0.14.0": {"x86_64-linux": {"tarball": "http://%s/missing.tar.xz", "shasum": "", "size": "0"}}}`, req.Host)
					_, _ = writer.Write([]byte(indexJSON)) //nolint:errcheck // test handler
				default:
					writer.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			oldURL := zigIndexDownloadURL
			defer func() { zigIndexDownloadURL = oldURL }()
			zigIndexDownloadURL = server.URL + "/index.json"

			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-download-error-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "0.14.0", tempDir)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Install", func() {
		It("installs from valid local tar.xz", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			archivePath := filepath.Join(downloadPath, "zig.tar.xz")
			createZigTarXz(archivePath)

			Expect(plugin.Install(context.Background(), "0.14.0", downloadPath, installPath)).To(Succeed())

			entries, err := os.ReadDir(installPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).NotTo(BeEmpty())
		})

		It("returns error when tarball doesn't exist", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-install-error-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			err = plugin.Install(context.Background(), "0.14.0", downloadPath, installPath)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when no directory is found in extracted archive", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "zig-install-nodir-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			archivePath := filepath.Join(downloadPath, "zig.tar.xz")
			createZigTarXzNoDir(archivePath)

			err = plugin.Install(context.Background(), "0.14.0", downloadPath, installPath)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("getPlatformKey", func() {
		It("returns a valid platform key", func() {
			plugin := New()
			key := plugin.getPlatformKey()
			Expect(key).NotTo(BeEmpty())

			Expect(key).To(MatchRegexp(`(x86_64|aarch64)-(linux|darwin)`))
		})
	})

	Describe("platformKeyFor", func() {
		DescribeTable("builds expected platform key",
			func(goos, arch, expected string) {
				key := platformKeyFor(goos, arch)
				Expect(key).To(Equal(expected))
			},
			Entry("amd64 linux", "linux", "amd64", "x86_64-linux"),
			Entry("arm64 linux", "linux", "arm64", "aarch64-linux"),
			Entry("other arch", "linux", "riscv64", "riscv64-linux"),
		)
	})

	Describe("copyDir", func() {
		It("copies directory contents", func() {
			tempDir, err := os.MkdirTemp("", "zig-copydir-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			srcDir := filepath.Join(tempDir, "src")
			dstDir := filepath.Join(tempDir, "dst")
			Expect(os.MkdirAll(srcDir, asdf.CommonDirectoryPermission)).To(Succeed())

			testFile := filepath.Join(srcDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test content"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			subDir := filepath.Join(srcDir, "subdir")
			Expect(os.MkdirAll(subDir, asdf.CommonDirectoryPermission)).To(Succeed())
			subFile := filepath.Join(subDir, "sub.txt")
			err = os.WriteFile(subFile, []byte("sub content"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = copyDir(srcDir, dstDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(dstDir, "test.txt"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(dstDir, "subdir", "sub.txt"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("copies symlinks", func() {
			tempDir, err := os.MkdirTemp("", "zig-copydir-symlink-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			srcDir := filepath.Join(tempDir, "src")
			dstDir := filepath.Join(tempDir, "dst")
			Expect(os.MkdirAll(srcDir, asdf.CommonDirectoryPermission)).To(Succeed())

			targetFile := filepath.Join(srcDir, "target.txt")
			Expect(os.WriteFile(targetFile, []byte("content"), asdf.CommonFilePermission)).To(Succeed())

			symlinkPath := filepath.Join(srcDir, "link.txt")
			Expect(os.Symlink("target.txt", symlinkPath)).To(Succeed())

			Expect(copyDir(srcDir, dstDir)).To(Succeed())

			copiedLink := filepath.Join(dstDir, "link.txt")
			info, err := os.Lstat(copiedLink)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModeSymlink).NotTo(BeZero())

			linkTarget, err := os.Readlink(copiedLink)
			Expect(err).NotTo(HaveOccurred())
			Expect(linkTarget).To(Equal("target.txt"))
		})
	})

	Describe("copyFile", func() {
		It("copies a file", func() {
			tempDir, err := os.MkdirTemp("", "zig-copyfile-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			srcFile := filepath.Join(tempDir, "src.txt")
			dstFile := filepath.Join(tempDir, "dst.txt")

			err = os.WriteFile(srcFile, []byte("test content"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = copyFile(srcFile, dstFile, asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(dstFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("test content"))
		})

		It("preserves executable permissions", func() {
			tempDir, err := os.MkdirTemp("", "zig-copyfile-exec-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			srcFile := filepath.Join(tempDir, "src")
			dstFile := filepath.Join(tempDir, "dst")

			err = os.WriteFile(srcFile, []byte("#!/bin/sh\necho test"), asdf.CommonDirectoryPermission)
			Expect(err).NotTo(HaveOccurred())

			err = copyFile(srcFile, dstFile, asdf.CommonDirectoryPermission)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(dstFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()&0o100).NotTo(BeZero(), "should be executable")
		})

		It("returns error when source file does not exist", func() {
			tempDir, err := os.MkdirTemp("", "zig-copyfile-missing-src-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			dstFile := filepath.Join(tempDir, "dst")
			err = copyFile(filepath.Join(tempDir, "missing"), dstFile, asdf.CommonFilePermission)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when destination cannot be created", func() {
			tempDir, err := os.MkdirTemp("", "zig-copyfile-bad-dst-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			srcFile := filepath.Join(tempDir, "src")
			Expect(os.WriteFile(srcFile, []byte("content"), asdf.CommonFilePermission)).To(Succeed())

			dstFile := filepath.Join(tempDir, "nonexistent", "dst")
			err = copyFile(srcFile, dstFile, asdf.CommonFilePermission)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Zig Plugin [online]", func() {
	Describe("ListAll (online)", func() {
		It("lists versions from the real Zig index when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Zig online ListAll test")
			}

			plugin := New()
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
		})
	})

	Describe("LatestStable (online)", func() {
		It("returns a latest stable version from the real Zig index when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Zig online LatestStable test")
			}

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
		})
	})
})

// createZigTarXzNoDir creates a tar.xz archive containing a single file at the root,
// without any top-level directory. This is used to exercise the error path in Install
// where no directory is found after extraction.
func createZigTarXzNoDir(archivePath string) {
	createZigTarXzWithName(archivePath, "zig")
}

// newZigIndexServer creates a mock Zig index server for tests.
func newZigIndexServer(index map[string]map[string]ZigRelease) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_, _ = w.Write([]byte(encodeZigIndex(index))) //nolint:errcheck // test helper
	}))
}

// encodeZigIndex encodes the minimal JSON structure expected by fetchIndex.
func encodeZigIndex(index map[string]map[string]ZigRelease) string {
	builder := &strings.Builder{}
	builder.WriteString("{")

	firstVersion := true
	for version, platforms := range index {
		if !firstVersion {
			builder.WriteString(",")
		}

		firstVersion = false

		builder.WriteString("\"")
		builder.WriteString(version)
		builder.WriteString("\":{")

		firstPlatform := true
		for platform, release := range platforms { //nolint:gocritic // it's fine, for a test helper
			if !firstPlatform {
				builder.WriteString(",")
			}

			firstPlatform = false

			builder.WriteString("\"")
			builder.WriteString(platform)
			builder.WriteString("\":{")
			builder.WriteString("\"tarball\":\"")
			builder.WriteString(release.Tarball)
			builder.WriteString("\",\"shasum\":\"")
			builder.WriteString(release.Shasum)
			builder.WriteString("\",\"size\":\"")
			builder.WriteString(release.Size)
			builder.WriteString("\"}")
		}

		builder.WriteString("}")
	}

	builder.WriteString("}")

	return builder.String()
}

// createZigTarXz creates a minimal tar.xz archive containing a single zig binary
// under a versioned top-level directory.
func createZigTarXz(archivePath string) {
	createZigTarXzWithName(archivePath, "zig-0.14.0/zig")
}

// createZigTarXzWithName creates a minimal tar.xz archive containing a single
// executable file with the provided header name.
func createZigTarXzWithName(archivePath, headerName string) {
	file, err := os.Create(archivePath)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	xzw, err := xz.NewWriter(file)
	Expect(err).NotTo(HaveOccurred())

	defer xzw.Close()

	tw := tar.NewWriter(xzw)
	defer tw.Close()

	content := []byte("#!/bin/sh\necho zig\n")
	header := &tar.Header{
		Name:     headerName,
		Mode:     int64(asdf.CommonDirectoryPermission),
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}

	Expect(tw.WriteHeader(header)).To(Succeed())

	_, err = tw.Write(content)
	Expect(err).NotTo(HaveOccurred())
}
