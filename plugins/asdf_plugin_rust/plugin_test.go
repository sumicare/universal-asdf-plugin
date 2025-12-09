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

package asdf_plugin_rust

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// errForcedRustHTTPClientError is returned by failingHTTPClient to force HTTP errors in tests.
var errForcedRustHTTPClientError = errors.New("forced HTTP client error")

// failingHTTPClient is an http.Client-like type used to force errors in tests.
type failingHTTPClient struct{}

// Do implements the HTTP client interface by always returning an error.
func (*failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, errForcedRustHTTPClientError
}

var _ = Describe("Rust Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("rust"))
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := New()
			Expect(plugin.Name()).To(Equal("rust"))
		})
	})

	Describe("ListBinPaths", func() {
		It("returns bin paths", func() {
			plugin := New()
			paths := plugin.ListBinPaths()
			Expect(paths).To(Equal("bin"))
		})
	})

	Describe("ExecEnv", func() {
		It("returns CARGO_HOME and RUSTUP_HOME", func() {
			plugin := New()
			env := plugin.ExecEnv("/tmp/install")
			Expect(env).To(HaveKey("CARGO_HOME"))
			Expect(env).To(HaveKey("RUSTUP_HOME"))
			Expect(env["CARGO_HOME"]).To(Equal("/tmp/install"))
			Expect(env["RUSTUP_HOME"]).To(Equal("/tmp/install"))
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns rust-toolchain files", func() {
			plugin := New()
			files := plugin.ListLegacyFilenames()
			Expect(files).To(ContainElements("rust-toolchain", "rust-toolchain.toml"))
		})
	})

	Describe("ParseLegacyFile", func() {
		It("parses plain rust-toolchain file", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-legacy-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			filePath := filepath.Join(tempDir, "rust-toolchain")
			err = os.WriteFile(filePath, []byte("1.75.0\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := plugin.ParseLegacyFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.75.0"))
		})

		It("parses rust-toolchain.toml file", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-legacy-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			filePath := filepath.Join(tempDir, "rust-toolchain.toml")
			content := `[toolchain]
channel = "1.75.0"
`
			err = os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := plugin.ParseLegacyFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.75.0"))
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-uninstall-*")
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Uninstall(context.Background(), tempDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tempDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("Help", func() {
		It("returns help information", func() {
			plugin := New()
			help := plugin.Help()
			Expect(help.Overview).To(ContainSubstring("Rust"))
			Expect(help.Links).To(ContainSubstring("rust-lang.org"))
		})
	})

	Describe("LatestStable", func() {
		It("returns stable for empty query", func() {
			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("stable"))
		})

		It("returns stable for stable query", func() {
			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "stable")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("stable"))
		})

		It("returns nightly for nightly query", func() {
			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "nightly")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("nightly"))
		})

		It("returns beta for beta query", func() {
			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "beta")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("beta"))
		})

		It("filters versions by query", func() {
			server := mock.NewServer("rust-lang", "rust")
			defer server.Close()
			server.RegisterTag("1.74.0")
			server.RegisterTag("1.75.0")
			server.RegisterTag("1.76.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()))
			version, err := plugin.LatestStable(context.Background(), "1.75")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.75.0"))
		})

		It("returns error when no versions match query", func() {
			server := mock.NewServer("rust-lang", "rust")
			defer server.Close()
			server.RegisterTag("1.74.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()))
			_, err := plugin.LatestStable(context.Background(), "9.")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions matching"))
		})

		It("propagates errors from ListAll in LatestStable", func() {
			failingClient := github.NewClientWithHTTP(&failingHTTPClient{}, "http://invalid-api")
			plugin := NewWithClient(failingClient)

			_, err := plugin.LatestStable(context.Background(), "1.75")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ListAll", func() {
		It("lists all versions from GitHub", func() {
			server := mock.NewServer("rust-lang", "rust")
			defer server.Close()
			server.RegisterTag("1.74.0")
			server.RegisterTag("1.75.0")
			server.RegisterTag("1.76.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()))
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("stable", "beta", "nightly", "1.74.0", "1.75.0", "1.76.0"))
		})

		It("filters out non-version tags", func() {
			server := mock.NewServer("rust-lang", "rust")
			defer server.Close()
			server.RegisterTag("1.74.0")
			server.RegisterTag("release-1.74.0")
			server.RegisterTag("1.75.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()))
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.74.0", "1.75.0"))
			Expect(versions).NotTo(ContainElement("release-1.74.0"))
		})

		It("propagates errors from the GitHub client in ListAll", func() {
			failingClient := github.NewClientWithHTTP(&failingHTTPClient{}, "http://invalid-api")
			plugin := NewWithClient(failingClient)

			versions, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(versions).To(BeEmpty())
		})
	})

	Describe("ParseLegacyFile errors", func() {
		It("returns error for toml without channel", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-legacy-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			filePath := filepath.Join(tempDir, "rust-toolchain.toml")
			content := `[toolchain]
profile = "minimal"
`
			err = os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			_, err = plugin.ParseLegacyFile(filePath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no channel found"))
		})

		It("returns error for non-existent file", func() {
			plugin := New()
			_, err := plugin.ParseLegacyFile("/non/existent/file")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download", func() {
		It("uses cached rustup-init script when already downloaded", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			scriptPath := filepath.Join(tempDir, "rustup-init.sh")

			content := make([]byte, 0, 2048)
			for i := range content {
				content = append(content, byte(i%256))
			}
			Expect(os.WriteFile(scriptPath, content, asdf.CommonDirectoryPermission)).To(Succeed())

			Expect(plugin.Download(context.Background(), "1.75.0", tempDir)).To(Succeed())
		})

		It("downloads rustup-init script from mock server", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("#!/bin/sh\necho rustup\n")) //nolint:errcheck // test handler
			}))
			defer server.Close()

			oldURL := rustupDownloadURL
			defer func() { rustupDownloadURL = oldURL }()
			rustupDownloadURL = server.URL

			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			Expect(plugin.Download(context.Background(), "1.75.0", tempDir)).To(Succeed())

			scriptPath := filepath.Join(tempDir, "rustup-init.sh")
			info, err := os.Stat(scriptPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()&asdf.ExecutablePermissionMask).NotTo(BeZero(), "script should be executable")

			versionFile := filepath.Join(tempDir, ".rust-version")
			versionData, err := os.ReadFile(versionFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(versionData)).To(Equal("1.75.0"))
		})

		It("returns error on download failure from mock server", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			oldURL := rustupDownloadURL
			defer func() { rustupDownloadURL = oldURL }()
			rustupDownloadURL = server.URL

			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-download-fail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.75.0", tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("download failed"))
		})

		It("returns error when rustupDownloadURL is malformed", func() {
			oldURL := rustupDownloadURL
			defer func() { rustupDownloadURL = oldURL }()
			rustupDownloadURL = "://bad-url"

			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-download-badurl-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.75.0", tempDir)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when HTTP request fails", func() {
			oldURL := rustupDownloadURL
			defer func() { rustupDownloadURL = oldURL }()
			rustupDownloadURL = "http://127.0.0.1:0"

			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-download-httperr-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.75.0", tempDir)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when script file cannot be created", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("#!/bin/sh\necho rustup\n")) //nolint:errcheck // test handler
			}))
			defer server.Close()

			oldURL := rustupDownloadURL
			defer func() { rustupDownloadURL = oldURL }()
			rustupDownloadURL = server.URL

			plugin := New()
			tempDir, err := os.MkdirTemp("", "rust-download-createfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			nonExistentDir := filepath.Join(tempDir, "nested", "dir")
			err = plugin.Download(context.Background(), "1.75.0", nonExistentDir)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Install", func() {
		It("runs rustup-init script with provided version", func() {
			plugin := New()
			baseDir, err := os.MkdirTemp("", "rust-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(baseDir)

			downloadDir := filepath.Join(baseDir, "download")
			installDir := filepath.Join(baseDir, "install")
			Expect(os.MkdirAll(downloadDir, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installDir, asdf.CommonDirectoryPermission)).To(Succeed())

			scriptPath := filepath.Join(downloadDir, "rustup-init.sh")
			script := []byte("#!/bin/sh\nexit 0\n")
			Expect(os.WriteFile(scriptPath, script, asdf.CommonDirectoryPermission)).To(Succeed())

			Expect(plugin.Install(context.Background(), "1.75.0", downloadDir, installDir)).To(Succeed())
		})

		It("returns error when rustup-init script is missing", func() {
			plugin := New()
			baseDir, err := os.MkdirTemp("", "rust-install-missing-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(baseDir)

			downloadDir := filepath.Join(baseDir, "download")
			installDir := filepath.Join(baseDir, "install")
			Expect(os.MkdirAll(downloadDir, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installDir, asdf.CommonDirectoryPermission)).To(Succeed())

			err = plugin.Install(context.Background(), "1.75.0", downloadDir, installDir)
			Expect(err).To(HaveOccurred())
		})
	})
})
