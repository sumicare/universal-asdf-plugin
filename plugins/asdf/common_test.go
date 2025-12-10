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
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Common package tests cover generic helpers that are reused by many
// concrete plugins (platform/arch detection, downloads and archive IO).
var _ = Describe("Common", func() {
	Describe("GetPlatform", func() {
		It("returns a valid platform string", func() {
			platform, err := GetPlatform()
			Expect(err).NotTo(HaveOccurred())
			Expect(platform).To(BeElementOf("linux", "darwin", "windows", "freebsd"))
		})
	})

	Describe("GetArch", func() {
		It("returns the current architecture", func() {
			arch, err := GetArch()
			Expect(err).NotTo(HaveOccurred())
			Expect(arch).NotTo(BeEmpty())
		})

		It("respects ASDF_OVERWRITE_ARCH", func() {
			original := os.Getenv("ASDF_OVERWRITE_ARCH")
			defer os.Setenv("ASDF_OVERWRITE_ARCH", original)

			os.Setenv("ASDF_OVERWRITE_ARCH", "arm64")
			arch, err := GetArch()
			Expect(err).NotTo(HaveOccurred())
			Expect(arch).To(Equal("arm64"))
		})

		DescribeTable("maps architectures correctly",
			func(input, expected string) {
				original := os.Getenv("ASDF_OVERWRITE_ARCH")
				defer os.Setenv("ASDF_OVERWRITE_ARCH", original)

				os.Setenv("ASDF_OVERWRITE_ARCH", input)
				arch, err := GetArch()
				Expect(err).NotTo(HaveOccurred())
				Expect(arch).To(Equal(expected))
			},
			Entry("amd64", "amd64", "amd64"),
			Entry("x86_64", "x86_64", "amd64"),
			Entry("386", "386", "386"),
			Entry("i386", "i386", "386"),
			Entry("i686", "i686", "386"),
			Entry("arm", "arm", "armv6l"),
			Entry("arm64", "arm64", "arm64"),
			Entry("aarch64", "aarch64", "arm64"),
			Entry("ppc64le", "ppc64le", "ppc64le"),
			Entry("loong64", "loong64", "loong64"),
			Entry("loongarch64", "loongarch64", "loong64"),
			Entry("riscv64", "riscv64", "riscv64"),
		)

		It("returns error for unsupported arch", func() {
			if runtime.GOARCH == "unsupported" {
				Skip("cannot test unsupported arch on this platform")
			}

			original := os.Getenv("ASDF_OVERWRITE_ARCH")
			defer os.Setenv("ASDF_OVERWRITE_ARCH", original)

			os.Setenv("ASDF_OVERWRITE_ARCH", "unsupported")
			_, err := GetArch()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("HTTPClient", func() {
		It("returns a configured HTTP client", func() {
			client := HTTPClient()
			Expect(client).NotTo(BeNil())
			Expect(client.Timeout).NotTo(BeZero())
		})
	})

	Describe("VerifySHA256", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "asdf-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("verifies correct checksum", func() {
			testFile := filepath.Join(tempDir, "test.txt")
			err := os.WriteFile(testFile, []byte("test content"), CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			expectedHash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
			err = VerifySHA256(testFile, expectedHash)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for incorrect checksum", func() {
			content := []byte("test content")
			filePath := filepath.Join(tempDir, "test.txt")
			err := os.WriteFile(filePath, content, CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = VerifySHA256(filePath, "wronghash")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("checksum mismatch"))
		})

		It("returns error for nonexistent file", func() {
			err := VerifySHA256("/nonexistent/file", "somehash")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("EnsureDir", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "asdf-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("creates nested directories", func() {
			nestedPath := filepath.Join(tempDir, "a", "b", "c")
			err := EnsureDir(nestedPath)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(nestedPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})

		It("succeeds if directory already exists", func() {
			err := EnsureDir(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("FilterVersions", func() {
		It("filters versions based on predicate", func() {
			versions := []string{"1.0.0", "1.1.0", "2.0.0", "2.1.0"}
			filtered := FilterVersions(versions, func(v string) bool {
				return v[0] == '1'
			})

			Expect(filtered).To(HaveLen(2))
			Expect(filtered).To(ContainElements("1.0.0", "1.1.0"))
		})

		It("returns empty slice if no matches", func() {
			versions := []string{"1.0.0", "1.1.0"}
			filtered := FilterVersions(versions, func(v string) bool {
				return v[0] == '3'
			})

			Expect(filtered).To(BeEmpty())
		})

		It("filters versions by predicate with prefix", func() {
			versions := []string{"1.20.0", "1.21.0", "1.21.5", "2.0.0"}
			filtered := FilterVersions(versions, func(v string) bool {
				return strings.HasPrefix(v, "1.21")
			})
			Expect(filtered).To(Equal([]string{"1.21.0", "1.21.5"}))
		})

		It("returns all versions when predicate always returns true", func() {
			versions := []string{"1.20.0", "1.21.0"}
			filtered := FilterVersions(versions, func(_ string) bool {
				return true
			})
			Expect(filtered).To(Equal(versions))
		})
	})

	Describe("SortVersions", func() {
		It("sorts versions in semver order", func() {
			versions := []string{"2.0.0", "1.0.0", "1.1.0", "10.0.0"}
			SortVersions(versions)
			Expect(versions).To(Equal([]string{"1.0.0", "1.1.0", "2.0.0", "10.0.0"}))
		})
	})

	Describe("CompareVersions", func() {
		DescribeTable("compares versions correctly",
			func(a, b string, expectedSign int) {
				result := CompareVersions(a, b)
				if expectedSign < 0 {
					Expect(result).To(BeNumerically("<", 0))
				} else if expectedSign > 0 {
					Expect(result).To(BeNumerically(">", 0))
				} else {
					Expect(result).To(Equal(0))
				}
			},
			Entry("1.0.0 < 2.0.0", "1.0.0", "2.0.0", -1),
			Entry("2.0.0 > 1.0.0", "2.0.0", "1.0.0", 1),
			Entry("1.0.0 == 1.0.0", "1.0.0", "1.0.0", 0),
			Entry("1.9 < 1.10", "1.9", "1.10", -1),
			Entry("1.21.0 > 1.20.0", "1.21.0", "1.20.0", 1),
		)
	})

	Describe("ParseVersionParts", func() {
		It("extracts numeric parts", func() {
			parts := ParseVersionParts("1.21.0")
			Expect(parts).To(Equal([]int{1, 21, 0}))
		})

		It("handles versions with prefixes", func() {
			parts := ParseVersionParts("go1.21.0")
			Expect(parts).To(Equal([]int{1, 21, 0}))
		})

		It("handles rc versions", func() {
			parts := ParseVersionParts("1.21rc1")
			Expect(parts).To(Equal([]int{1, 21, 1}))
		})
	})

	Describe("ReadLegacyVersionFile", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "asdf-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("reads and trims version file", func() {
			filePath := filepath.Join(tempDir, ".version")
			err := os.WriteFile(filePath, []byte("  1.21.0  \n"), CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := ReadLegacyVersionFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.21.0"))
		})

		It("returns error for nonexistent file", func() {
			_, err := ReadLegacyVersionFile("/nonexistent/file")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Msg and Err", func() {
		It("does not panic", func() {
			Expect(func() { Msgf("test %s", "message") }).NotTo(Panic())
			Expect(func() { Errf("test %s", "error") }).NotTo(Panic())
		})
	})

	Describe("IsOnline", func() {
		var originalValue string

		BeforeEach(func() {
			originalValue = os.Getenv("ONLINE")
		})

		AfterEach(func() {
			os.Setenv("ONLINE", originalValue)
		})

		DescribeTable("returns correct value",
			func(envValue string, expected bool) {
				os.Setenv("ONLINE", envValue)
				Expect(IsOnline()).To(Equal(expected))
			},
			Entry("1 is true", "1", true),
			Entry("true is true", "true", true),
			Entry("TRUE is true", "TRUE", true),
			Entry("empty is false", "", false),
			Entry("0 is false", "0", false),
			Entry("false is false", "false", false),
		)
	})

	Describe("DownloadFile", func() {
		var tempDir string
		var server *httptest.Server

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "download-test-*")
			Expect(err).NotTo(HaveOccurred())

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/test.txt" {
					_, _ = w.Write([]byte("test content")) //nolint:errcheck // response writer errors are ignored in test server
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
			server.Close()
		})

		It("downloads file successfully", func() {
			destPath := filepath.Join(tempDir, "downloaded.txt")
			err := DownloadFile(context.Background(), server.URL+"/test.txt", destPath)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("test content"))
		})

		It("returns error for 404", func() {
			destPath := filepath.Join(tempDir, "notfound.txt")
			err := DownloadFile(context.Background(), server.URL+"/notfound", destPath)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid URL", func() {
			destPath := filepath.Join(tempDir, "invalid.txt")
			err := DownloadFile(context.Background(), "http://invalid.invalid.invalid:99999/file", destPath)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when cannot create destination file", func() {
			err := DownloadFile(context.Background(), server.URL+"/test.txt", "/nonexistent/path/file.txt")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DownloadString", func() {
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/content" {
					n, err := w.Write([]byte("string content"))
					Expect(err).NotTo(HaveOccurred())
					Expect(n).To(Equal(len("string content")))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
		})

		AfterEach(func() {
			server.Close()
		})

		It("downloads string successfully", func() {
			content, err := DownloadString(context.Background(), server.URL+"/content")
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal("string content"))
		})

		It("returns error for 404", func() {
			_, err := DownloadString(context.Background(), server.URL+"/notfound")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid URL", func() {
			_, err := DownloadString(context.Background(), "http://invalid.invalid.invalid:99999/content")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("LatestVersion", func() {
		It("returns latest version", func() {
			versions := []string{"1.0.0", "2.0.0", "1.5.0"}
			latest := LatestVersion(versions, "")
			Expect(latest).To(Equal("2.0.0"))
		})

		It("returns latest matching pattern", func() {
			versions := []string{"1.0.0", "2.0.0", "1.5.0"}
			latest := LatestVersion(versions, "1")
			Expect(latest).To(Equal("1.5.0"))
		})

		It("prefers stable versions over prereleases", func() {
			versions := []string{"1.0.0", "1.1.0-rc1", "1.1.0"}
			latest := LatestVersion(versions, "")
			Expect(latest).To(Equal("1.1.0"))
		})

		It("falls back to prereleases when no stable versions exist", func() {
			versions := []string{"1.1.0-rc1", "1.1.0-beta1"}
			latest := LatestVersion(versions, "")
			Expect(latest).To(Equal("1.1.0-beta1"))
		})

		It("returns empty string if no match", func() {
			versions := []string{"1.0.0", "2.0.0"}
			latest := LatestVersion(versions, "3")
			Expect(latest).To(BeEmpty())
		})

		It("returns empty string for empty list", func() {
			latest := LatestVersion(make([]string, 0), "")
			Expect(latest).To(BeEmpty())
		})
	})
})
