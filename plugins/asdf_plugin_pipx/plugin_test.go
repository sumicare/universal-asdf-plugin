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

package asdf_plugin_pipx

import (
	"bytes"
	"context"
	"errors"
	"io"
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

var (
	// errForcedHTTPError simulates a failing HTTP client for pipx tests.
	errForcedHTTPError = errors.New("forced HTTP error")
	// errRequestCreationFailed simulates a request construction failure in pipx tests.
	errRequestCreationFailed = errors.New("request creation failed")
	// errEnsureToolchainsFailed simulates a toolchain ensure failure in pipx tests.
	errEnsureToolchainsFailed = errors.New("ensure toolchains failed")
	// errCopyFailed simulates a file copy failure in pipx tests.
	errCopyFailed = errors.New("copy failed")
	// errWriteFailed simulates a wrapper script write failure in pipx tests.
	errWriteFailed = errors.New("write failed")
)

// failingHTTPClient is an http.Client-like type that always returns an error.
type failingHTTPClient struct{}

// Do implements the minimal http.Client interface and always fails for tests.
func (*failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, errForcedHTTPError
}

// failingReadBody simulates a response body that fails during Read.
type failingReadBody struct{}

// Read always returns an error to simulate a write failure in Download.
func (*failingReadBody) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// Close is a no-op implementation to satisfy io.ReadCloser.
func (*failingReadBody) Close() error {
	return nil
}

// failingDownloadHTTPClient returns a response with a failing body for tests.
type failingDownloadHTTPClient struct{}

// Do returns an HTTP 200 response whose body fails when read.
func (*failingDownloadHTTPClient) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       &failingReadBody{},
	}, nil
}

var _ = Describe("Pipx Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("pipx"))
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := New()
			Expect(plugin.Name()).To(Equal("pipx"))
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
		It("returns nil environment", func() {
			plugin := New()
			env := plugin.ExecEnv("/tmp/install")
			Expect(env).To(BeNil())
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns nil", func() {
			plugin := New()
			files := plugin.ListLegacyFilenames()
			Expect(files).To(BeNil())
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "pipx-uninstall-*")
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
			Expect(help.Overview).To(ContainSubstring("pipx"))
			Expect(help.Links).To(ContainSubstring("pipx.pypa.io"))
		})
	})

	Describe("ParseLegacyFile", func() {
		It("reads version from file", func() {
			plugin := New()
			tempFile, err := os.CreateTemp("", "pipx-version-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString("1.2.3\n")
			Expect(err).NotTo(HaveOccurred())
			tempFile.Close()

			version, err := plugin.ParseLegacyFile(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.3"))
		})
	})

	Describe("ListAll", func() {
		It("lists all versions from GitHub", func() {
			server := mock.NewServer("pypa", "pipx")
			defer server.Close()
			server.RegisterTag("1.0.0")
			server.RegisterTag("1.1.0")
			server.RegisterTag("1.2.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()), pipxDownloadURL)
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.0.0", "1.1.0", "1.2.0"))
		})

		It("filters out non-semver versions", func() {
			server := mock.NewServer("pypa", "pipx")
			defer server.Close()
			server.RegisterTag("1.0.0")
			server.RegisterTag("invalid")
			server.RegisterTag("1.1.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()), pipxDownloadURL)
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.0.0", "1.1.0"))
			Expect(versions).NotTo(ContainElement("invalid"))
		})

		It("lists pipx versions from GitHub when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for pipx GitHub ListAll test")
			}

			plugin := New()
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
		})

		It("propagates errors from the GitHub client", func() {
			client := github.NewClientWithHTTP(&failingHTTPClient{}, "http://invalid-api")
			plugin := NewWithClient(client, pipxDownloadURL)

			versions, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(versions).To(BeEmpty())
		})
	})

	Describe("LatestStable", func() {
		It("returns latest version", func() {
			server := mock.NewServer("pypa", "pipx")
			defer server.Close()
			server.RegisterTag("1.0.0")
			server.RegisterTag("1.1.0")
			server.RegisterTag("1.2.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()), pipxDownloadURL)
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.0"))
		})

		It("filters by query", func() {
			server := mock.NewServer("pypa", "pipx")
			defer server.Close()
			server.RegisterTag("1.0.0")
			server.RegisterTag("1.1.0")
			server.RegisterTag("2.0.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()), pipxDownloadURL)
			version, err := plugin.LatestStable(context.Background(), "1.")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.1.0"))
		})

		It("returns error when no versions match query", func() {
			server := mock.NewServer("pypa", "pipx")
			defer server.Close()
			server.RegisterTag("1.0.0")

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()), pipxDownloadURL)
			_, err := plugin.LatestStable(context.Background(), "9.")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions matching"))
		})

		It("returns error when no versions found", func() {
			server := mock.NewServer("pypa", "pipx")
			defer server.Close()

			plugin := NewWithClient(github.NewClientWithHTTP(server.Client(), server.URL()), pipxDownloadURL)
			_, err := plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions found"))
		})

		It("propagates errors from ListAll", func() {
			client := github.NewClientWithHTTP(&failingHTTPClient{}, "http://invalid-api")
			plugin := NewWithClient(client, pipxDownloadURL)

			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(version).To(BeEmpty())
		})

		It("returns latest stable version from GitHub when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for pipx GitHub LatestStable test")
			}

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
		})
	})

	Describe("Download", func() {
		It("downloads pipx.pyz file", func() {
			downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("pyz content")) //nolint:errcheck // test mock
			}))
			defer downloadServer.Close()

			plugin := NewWithClient(nil, downloadServer.URL+"/%s/pipx.pyz")

			tempDir, err := os.MkdirTemp("", "pipx-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).NotTo(HaveOccurred())

			pyzPath := filepath.Join(tempDir, "pipx.pyz")
			Expect(pyzPath).To(BeAnExistingFile())

			content, err := os.ReadFile(pyzPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("pyz content"))
		})

		It("uses cached download when pipx.pyz already exists and is large enough", func() {
			var downloadCalled bool

			downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				downloadCalled = true
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer downloadServer.Close()

			plugin := NewWithClient(nil, downloadServer.URL+"/%s/pipx.pyz")

			tempDir, err := os.MkdirTemp("", "pipx-download-cache-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			pyzPath := filepath.Join(tempDir, "pipx.pyz")
			Expect(os.WriteFile(pyzPath, bytes.Repeat([]byte("x"), 2048), asdf.CommonFilePermission)).To(Succeed())

			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(downloadCalled).To(BeFalse(), "Download should use cached file and not hit server")
		})

		It("returns error on download failure", func() {
			downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer downloadServer.Close()

			plugin := NewWithClient(nil, downloadServer.URL+"/%s/pipx.pyz")

			tempDir, err := os.MkdirTemp("", "pipx-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("download failed"))
		})

		It("returns error when request creation fails", func() {
			originalNewRequestFn := newRequestFn
			newRequestFn = func(context.Context, string, string, io.Reader) (*http.Request, error) {
				return nil, errRequestCreationFailed
			}
			defer func() { newRequestFn = originalNewRequestFn }()

			plugin := New()

			tempDir, err := os.MkdirTemp("", "pipx-download-reqfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadErr := plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(downloadErr).To(HaveOccurred())
			Expect(downloadErr.Error()).To(ContainSubstring("creating request"))
		})

		It("returns error when HTTP client fails", func() {
			originalClient := httpClient
			httpClient = &failingHTTPClient{}
			defer func() { httpClient = originalClient }()

			plugin := NewWithClient(nil, "https://invalid.local/%s/pipx.pyz")

			tempDir, err := os.MkdirTemp("", "pipx-download-clientfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadErr := plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(downloadErr).To(HaveOccurred())
			Expect(downloadErr.Error()).To(ContainSubstring("downloading pipx"))
		})

		It("returns error when writing downloaded file fails", func() {
			originalClient := httpClient
			httpClient = &failingDownloadHTTPClient{}
			defer func() { httpClient = originalClient }()

			plugin := NewWithClient(nil, "https://invalid.local/%s/pipx.pyz")

			tempDir, err := os.MkdirTemp("", "pipx-download-writefail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadErr := plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(downloadErr).To(HaveOccurred())
			Expect(downloadErr.Error()).To(ContainSubstring("writing file"))
		})
	})

	Describe("Install", func() {
		var originalInstallPythonToolchain func(context.Context) error
		var originalEnsureToolchains func(context.Context, ...string) error
		var originalCopyFileFn func(io.Writer, io.Reader) (int64, error)
		var originalWriteFileFn func(string, []byte, os.FileMode) error

		BeforeEach(func() {
			originalInstallPythonToolchain = installPythonToolchain
			installPythonToolchain = func(_ context.Context) error {
				return nil
			}

			originalEnsureToolchains = ensureToolchains
			ensureToolchains = func(_ context.Context, _ ...string) error {
				return nil
			}

			originalCopyFileFn = copyFileFn
			copyFileFn = io.Copy

			originalWriteFileFn = writeFileFn
			writeFileFn = os.WriteFile
		})

		AfterEach(func() {
			installPythonToolchain = originalInstallPythonToolchain
			ensureToolchains = originalEnsureToolchains
			copyFileFn = originalCopyFileFn
			writeFileFn = originalWriteFileFn
		})

		It("installs pipx from downloaded pyz file", func() {
			plugin := New()

			tempDir, err := os.MkdirTemp("", "pipx-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())

			pyzPath := filepath.Join(downloadPath, "pipx.pyz")
			Expect(os.WriteFile(pyzPath, []byte("pyz content"), asdf.CommonFilePermission)).To(Succeed())

			err = plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(err).NotTo(HaveOccurred())

			installedPyz := filepath.Join(installPath, "bin", "pipx.pyz")
			Expect(installedPyz).To(BeAnExistingFile())

			wrapperPath := filepath.Join(installPath, "bin", "pipx")
			Expect(wrapperPath).To(BeAnExistingFile())

			wrapperContent, err := os.ReadFile(wrapperPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(wrapperContent)).To(ContainSubstring("python3"))
			Expect(string(wrapperContent)).To(ContainSubstring("pipx.pyz"))
		})

		It("returns error when ensuring toolchains fails", func() {
			ensureToolchains = func(_ context.Context, _ ...string) error {
				return errEnsureToolchainsFailed
			}

			plugin := New()

			tempDir, err := os.MkdirTemp("", "pipx-install-toolchain-fail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			installPath := filepath.Join(tempDir, "install")
			downloadPath := filepath.Join(tempDir, "download")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())

			installErr := plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("ensure toolchains failed"))
		})

		It("returns error when pipx.pyz source file is missing", func() {
			plugin := New()

			tempDir, err := os.MkdirTemp("", "pipx-install-missing-src-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())

			installErr := plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("opening source file"))
		})

		It("returns error when bin directory cannot be created", func() {
			plugin := New()

			tempDir, err := os.MkdirTemp("", "pipx-install-mkdirfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())
			Expect(os.MkdirAll(installPath, 0o755)).To(Succeed())

			binPath := filepath.Join(installPath, "bin")
			Expect(os.WriteFile(binPath, []byte("not a dir"), asdf.CommonFilePermission)).To(Succeed())

			installErr := plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("creating bin directory"))
		})

		It("returns error when destination file cannot be created", func() {
			plugin := New()

			tempDir, err := os.MkdirTemp("", "pipx-install-dstfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())
			Expect(os.MkdirAll(installPath, 0o755)).To(Succeed())

			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())
			srcPath := filepath.Join(downloadPath, "pipx.pyz")
			Expect(os.WriteFile(srcPath, []byte("pyz content"), asdf.CommonFilePermission)).To(Succeed())

			binDir := filepath.Join(installPath, "bin")
			Expect(os.MkdirAll(binDir, 0o755)).To(Succeed())
			dstPath := filepath.Join(binDir, "pipx.pyz")
			Expect(os.MkdirAll(dstPath, 0o755)).To(Succeed())

			installErr := plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("creating destination file"))
		})

		It("returns error when copying pipx.pyz fails", func() {
			plugin := New()

			copyFileFn = func(io.Writer, io.Reader) (int64, error) {
				return 0, errCopyFailed
			}

			tempDir, err := os.MkdirTemp("", "pipx-install-copyfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())
			Expect(os.MkdirAll(installPath, 0o755)).To(Succeed())

			srcPath := filepath.Join(downloadPath, "pipx.pyz")
			Expect(os.WriteFile(srcPath, []byte("pyz content"), asdf.CommonFilePermission)).To(Succeed())

			installErr := plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("copying file"))
		})

		It("returns error when wrapper script cannot be written", func() {
			plugin := New()

			writeFileFn = func(string, []byte, os.FileMode) error {
				return errWriteFailed
			}

			tempDir, err := os.MkdirTemp("", "pipx-install-wrapperfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, 0o755)).To(Succeed())
			Expect(os.MkdirAll(installPath, 0o755)).To(Succeed())

			srcPath := filepath.Join(downloadPath, "pipx.pyz")
			Expect(os.WriteFile(srcPath, []byte("pyz content"), asdf.CommonFilePermission)).To(Succeed())

			installErr := plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("creating wrapper script"))
		})
	})
})
