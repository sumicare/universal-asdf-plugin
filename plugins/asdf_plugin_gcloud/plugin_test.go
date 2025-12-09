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

package asdf_plugin_gcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// errForcedGcloudHTTPClientError simulates a failing HTTP client in gcloud tests.
var errForcedGcloudHTTPClientError = errors.New("forced http client error")

// errForcedGcloudReadError is returned by failingReadCloser to force read failures.
var errForcedGcloudReadError = errors.New("forced read error")

// errForcedGcloudExtractError simulates a failure in archive extraction.
var errForcedGcloudExtractError = errors.New("forced extract error")

// errForcedGcloudToolchainError simulates a failure while ensuring toolchains.
var errForcedGcloudToolchainError = errors.New("forced toolchain error")

// errForcedGcloudPythonInstallError simulates a failure while installing Python.
var errForcedGcloudPythonInstallError = errors.New("forced python install error")

// roundTripperFunc adapts a function so it satisfies http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper by delegating to the underlying function.
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// failingReadCloser is an io.ReadCloser that always fails, used to force io.Copy errors.
type failingReadCloser struct{}

// Read implements io.ReadCloser by always returning an error.
func (*failingReadCloser) Read(_ []byte) (int, error) {
	return 0, errForcedGcloudReadError
}

// Close implements io.ReadCloser and is a no-op for failingReadCloser.
func (*failingReadCloser) Close() error { return nil }

var _ = Describe("Gcloud Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("gcloud"))
		})
	})

	Describe("NewWithURL", func() {
		It("creates a plugin with custom URL", func() {
			plugin := NewWithURL("http://test.example.com")
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.apiURL).To(Equal("http://test.example.com"))
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := New()
			Expect(plugin.Name()).To(Equal("gcloud"))
		})
	})

	Describe("ListBinPaths", func() {
		It("returns bin paths", func() {
			plugin := New()
			paths := plugin.ListBinPaths()
			Expect(paths).To(Equal("google-cloud-sdk/bin"))
		})
	})

	Describe("ExecEnv", func() {
		It("returns CLOUDSDK_ROOT_DIR", func() {
			plugin := New()
			env := plugin.ExecEnv("/tmp/install")
			Expect(env).To(HaveKey("CLOUDSDK_ROOT_DIR"))
			Expect(env["CLOUDSDK_ROOT_DIR"]).To(Equal(filepath.Clean("/tmp/install/google-cloud-sdk")))
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns nil", func() {
			plugin := New()
			files := plugin.ListLegacyFilenames()
			Expect(files).To(BeNil())
		})
	})

	Describe("ParseLegacyFile", func() {
		It("parses version from file", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "gcloud-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			versionFile := filepath.Join(tempDir, ".gcloud-version")
			err = os.WriteFile(versionFile, []byte("548.0.0\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := plugin.ParseLegacyFile(versionFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("548.0.0"))
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "gcloud-uninstall-*")
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
			Expect(help.Overview).To(ContainSubstring("Google Cloud SDK"))
			Expect(help.Links).To(ContainSubstring("cloud.google.com"))
		})
	})

	Describe("ListAll", func() {
		var fixture *gcloudTestFixture

		BeforeEach(func() {
			fixture = newGcloudTestFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("lists gcloud versions", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			versions, err := fixture.plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			goldieVersions, err := fixture.GoldieVersions()
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements(goldieVersions))
		})

		It("lists gcloud versions from GCS when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for gcloud GCS ListAll test")
			}

			plugin := New()
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
		})

		It("returns error when request cannot be created", func() {
			plugin := NewWithURL("://bad-url")

			versions, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(versions).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("creating request"))
		})

		It("returns error when response cannot be decoded", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := w.Write([]byte("not-json")); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}))
			defer server.Close()

			originalClient := gcloudHTTPClient
			gcloudHTTPClient = server.Client()
			defer func() { gcloudHTTPClient = originalClient }()

			plugin := NewWithURL(server.URL)
			versions, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(versions).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("decoding response"))
		})

		It("handles paginated responses and de-duplicates versions", func() {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				callCount++

				writer.Header().Set("Content-Type", "application/json")

				var resp gcsResponse
				if callCount == 1 {
					resp = gcsResponse{
						NextPageToken: "next-page-token",
						Items: []gcsObject{
							{Name: "google-cloud-sdk-548.0.0-linux-x86_64.tar.gz"},
						},
					}
				} else {
					resp = gcsResponse{
						Items: []gcsObject{
							{Name: "google-cloud-sdk-548.0.0-linux-x86_64.tar.gz"},
							{Name: "google-cloud-sdk-549.0.0-linux-x86_64.tar.gz"},
						},
					}
				}

				_ = json.NewEncoder(writer).Encode(resp) //nolint:errcheck // test mock
			}))
			defer server.Close()

			originalClient := gcloudHTTPClient
			gcloudHTTPClient = server.Client()
			defer func() { gcloudHTTPClient = originalClient }()

			plugin := NewWithURL(server.URL)
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("548.0.0", "549.0.0"))
		})
	})

	Describe("LatestStable", func() {
		var fixture *gcloudTestFixture

		BeforeEach(func() {
			fixture = newGcloudTestFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns latest stable version", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())

			goldieLatest, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(goldieLatest))
		})

		It("returns latest stable version from GCS when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for gcloud GCS LatestStable test")
			}

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
			Expect(strings.Count(version, ".")).To(BeNumerically(">=", 2))
		})

		It("filters by pattern", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			filterPattern, err := fixture.GoldieFilterPattern()
			Expect(err).NotTo(HaveOccurred())

			version, err := fixture.plugin.LatestStable(context.Background(), filterPattern)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
		})

		It("returns error when no versions match", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			_, err := fixture.plugin.LatestStable(context.Background(), "nonexistent-version-pattern")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions matching"))
		})

		It("returns error when no versions are available", func() {
			fixture.Close()
			fixture = newGcloudTestFixture()
			fixture.SetupVersions(nil)

			_, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions found"))
		})

		It("propagates errors from ListAll", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := w.Write([]byte("not-json")); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}))
			defer server.Close()

			originalClient := gcloudHTTPClient
			gcloudHTTPClient = server.Client()
			defer func() { gcloudHTTPClient = originalClient }()

			plugin := NewWithURL(server.URL)
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(version).To(BeEmpty())
		})
	})

	Describe("getObjectName", func() {
		It("returns correct object name for current platform", func() {
			plugin := New()

			testdataPath := testutil.GoldieTestDataPath(GinkgoT())
			testVersion, err := testutil.ReadGoldieLatest(testdataPath, "gcloud_latest_stable.golden")
			if err != nil {
				testVersion = "548.0.0"
			}

			name, err := plugin.getObjectName(testVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(ContainSubstring("google-cloud-sdk-" + testVersion))
			Expect(name).To(ContainSubstring(".tar.gz"))
		})

		It("returns linux amd64 object name", func() {
			plugin := newPluginWithRuntime("https://example.invalid", gcloudRuntime{goos: "linux", arch: "amd64"})

			name, err := plugin.getObjectName("548.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("google-cloud-sdk-548.0.0-linux-x86_64.tar.gz"))
		})

		It("returns linux arm64 object name", func() {
			plugin := newPluginWithRuntime("https://example.invalid", gcloudRuntime{goos: "linux", arch: "arm64"})

			name, err := plugin.getObjectName("548.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("google-cloud-sdk-548.0.0-linux-arm.tar.gz"))
		})

		It("returns darwin amd64 object name", func() {
			plugin := newPluginWithRuntime("https://example.invalid", gcloudRuntime{goos: "darwin", arch: "amd64"})

			name, err := plugin.getObjectName("548.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("google-cloud-sdk-548.0.0-darwin-x86_64.tar.gz"))
		})

		It("returns darwin arm64 object name", func() {
			plugin := newPluginWithRuntime("https://example.invalid", gcloudRuntime{goos: "darwin", arch: "arm64"})

			name, err := plugin.getObjectName("548.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("google-cloud-sdk-548.0.0-darwin-arm.tar.gz"))
		})

		It("returns error for unsupported arch", func() {
			plugin := newPluginWithRuntime("https://example.invalid", gcloudRuntime{goos: "linux", arch: "ppc64"})

			name, err := plugin.getObjectName("548.0.0")
			Expect(err).To(HaveOccurred())
			Expect(name).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("unsupported architecture"))
		})

		It("returns error for unsupported OS", func() {
			plugin := newPluginWithRuntime("https://example.invalid", gcloudRuntime{goos: "windows", arch: "amd64"})

			name, err := plugin.getObjectName("548.0.0")
			Expect(err).To(HaveOccurred())
			Expect(name).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("unsupported platform"))
		})
	})

	Describe("Download", func() {
		It("uses cached download when archive already exists", func() {
			plugin := New()

			version := "548.0.0"
			objectName, err := plugin.getObjectName(version)
			Expect(err).NotTo(HaveOccurred())

			downloadDir, err := os.MkdirTemp("", "gcloud-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			filePath := filepath.Join(downloadDir, objectName)

			content := make([]byte, 0, 2048)
			for i := range content {
				content = append(content, byte(i%256))
			}
			Expect(os.WriteFile(filePath, content, asdf.CommonFilePermission)).To(Succeed())

			Expect(plugin.Download(context.Background(), version, downloadDir)).To(Succeed())
		})

		Describe("with mocked HTTP client", func() {
			var (
				fixture            *gcloudTestFixture
				server             *httptest.Server
				originalHTTPClient *http.Client
				originalBaseURL    string
			)

			BeforeEach(func() {
				fixture = newGcloudTestFixture()
				originalHTTPClient = gcloudHTTPClient
				originalBaseURL = gcloudDownloadBaseURL
			})

			AfterEach(func() {
				gcloudHTTPClient = originalHTTPClient
				gcloudDownloadBaseURL = originalBaseURL
				if server != nil {
					server.Close()
				}
				fixture.Close()
			})

			It("downloads gcloud archive using a mocked HTTP client (always on)", func() {
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}
				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

				version, err := fixture.plugin.LatestStable(context.Background(), "")
				Expect(err).NotTo(HaveOccurred())

				downloadDir, err := os.MkdirTemp("", "gcloud-download-mock-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					if _, err = writer.Write([]byte("mock gcloud archive")); err != nil {
						http.Error(writer, err.Error(), http.StatusInternalServerError)
					}
				}))

				gcloudDownloadBaseURL = server.URL
				gcloudHTTPClient = server.Client()

				plugin := New()
				err = plugin.Download(context.Background(), version, downloadDir)
				Expect(err).NotTo(HaveOccurred())

				files, err := os.ReadDir(downloadDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(files).NotTo(BeEmpty())
			})

			It("downloads gcloud archive against local server when ONLINE=1", func() {
				if !asdf.IsOnline() {
					Skip("ONLINE=1 required for gcloud Download test")
				}

				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}
				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

				version, err := fixture.plugin.LatestStable(context.Background(), "")
				Expect(err).NotTo(HaveOccurred())

				downloadDir, err := os.MkdirTemp("", "gcloud-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					if _, err := writer.Write([]byte("mock gcloud archive")); err != nil {
						http.Error(writer, err.Error(), http.StatusInternalServerError)
					}
				}))

				gcloudDownloadBaseURL = server.URL
				gcloudHTTPClient = server.Client()

				plugin := New()
				err = plugin.Download(context.Background(), version, downloadDir)
				Expect(err).NotTo(HaveOccurred())

				files, err := os.ReadDir(downloadDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(files).NotTo(BeEmpty())
			})
		})

		It("returns error when HTTP status is not OK", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			downloadDir, err := os.MkdirTemp("", "gcloud-download-status-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusInternalServerError)
			}))

			gcloudDownloadBaseURL = server.URL
			gcloudHTTPClient = server.Client()

			plugin := New()
			err = plugin.Download(context.Background(), version, downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("download failed"))
		})

		It("returns error when HTTP client fails", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			downloadDir, err := os.MkdirTemp("", "gcloud-download-http-error-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			gcloudDownloadBaseURL = "https://gcloud.invalid.test"
			originalClient := gcloudHTTPClient
			gcloudHTTPClient = &http.Client{
				Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, errForcedGcloudHTTPClientError
				}),
			}
			defer func() { gcloudHTTPClient = originalClient }()

			plugin := New()
			err = plugin.Download(context.Background(), version, downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("downloading gcloud"))
		})

		It("returns error when output file cannot be created", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			downloadDir, err := os.MkdirTemp("", "gcloud-download-openfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			Expect(os.Chmod(downloadDir, 0o555)).To(Succeed())

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if _, err := writer.Write([]byte("mock gcloud archive")); err != nil {
					http.Error(writer, err.Error(), http.StatusInternalServerError)
				}
			}))

			gcloudDownloadBaseURL = server.URL
			gcloudHTTPClient = server.Client()

			plugin := New()
			plugin.runtime.goos = runtime.GOOS
			plugin.runtime.arch = runtime.GOARCH

			err = plugin.Download(context.Background(), version, downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating file"))
		})

		It("returns error when request cannot be created", func() {
			plugin := newPluginWithRuntime(fmt.Sprintf(gcsAPIURL, gcsBucketName), gcloudRuntime{goos: "linux", arch: "amd64"})

			savedBaseURL := gcloudDownloadBaseURL
			defer func() { gcloudDownloadBaseURL = savedBaseURL }()

			gcloudDownloadBaseURL = "://bad-url"

			downloadDir, err := os.MkdirTemp("", "gcloud-download-badrequest-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			err = plugin.Download(context.Background(), "548.0.0", downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating request"))
		})

		It("returns error when platform is unsupported", func() {
			plugin := newPluginWithRuntime(fmt.Sprintf(gcsAPIURL, gcsBucketName), gcloudRuntime{goos: "windows", arch: "amd64"})

			downloadDir, err := os.MkdirTemp("", "gcloud-download-unsupported-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			err = plugin.Download(context.Background(), "548.0.0", downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported platform"))
		})

		It("returns error when writing downloaded file fails", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			downloadDir, err := os.MkdirTemp("", "gcloud-download-writefail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			originalClient := gcloudHTTPClient
			gcloudHTTPClient = &http.Client{
				Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       &failingReadCloser{},
					}, nil
				}),
			}
			defer func() { gcloudHTTPClient = originalClient }()

			plugin := New()
			plugin.runtime.goos = runtime.GOOS
			plugin.runtime.arch = runtime.GOARCH

			err = plugin.Download(context.Background(), version, downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("writing file"))
		})
	})

	Describe("Install", func() {
		var (
			originalEnsureToolchains       func(context.Context, ...string) error
			originalInstallPythonToolchain func(context.Context) error
		)

		BeforeEach(func() {
			originalEnsureToolchains = ensureToolchains
			ensureToolchains = func(_ context.Context, _ ...string) error {
				return nil
			}

			originalInstallPythonToolchain = installPythonToolchain
			installPythonToolchain = func(_ context.Context) error {
				return nil
			}
		})

		AfterEach(func() {
			ensureToolchains = originalEnsureToolchains
			installPythonToolchain = originalInstallPythonToolchain
		})

		It("returns error when archive is missing", func() {
			plugin := New()

			version := "548.0.0"
			downloadDir, err := os.MkdirTemp("", "gcloud-install-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "gcloud-install-target-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			err = plugin.Install(context.Background(), version, downloadDir, installDir)
			Expect(err).To(HaveOccurred())
		})

		Context("with mocked extractor", func() {
			var originalExtractTarGz func(string, string) error

			BeforeEach(func() {
				originalExtractTarGz = gcloudExtractTarGz
			})

			AfterEach(func() {
				gcloudExtractTarGz = originalExtractTarGz
			})

			It("installs gcloud using mocked dependencies", func() {
				var ensuredTools []string
				ensureToolchains = func(_ context.Context, tools ...string) error {
					ensuredTools = append(ensuredTools, tools...)
					return nil
				}

				version := "548.0.0"
				plugin := New()

				downloadDir, err := os.MkdirTemp("", "gcloud-install-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "gcloud-install-target-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				objectName, err := plugin.getObjectName(version)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, objectName)
				Expect(os.WriteFile(archivePath, []byte("mock archive"), asdf.CommonFilePermission)).To(Succeed())

				gcloudExtractTarGz = func(archive, dest string) error {
					Expect(archive).To(Equal(archivePath))

					return os.MkdirAll(filepath.Join(dest, "google-cloud-sdk"), asdf.CommonDirectoryPermission)
				}

				err = plugin.Install(context.Background(), version, downloadDir, installDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(ensuredTools).To(ContainElement("python"))
			})

			It("returns error when extracting archive fails", func() {
				version := "548.0.0"
				plugin := New()

				downloadDir, err := os.MkdirTemp("", "gcloud-install-download-extractfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "gcloud-install-target-extractfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				objectName, err := plugin.getObjectName(version)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, objectName)
				Expect(os.WriteFile(archivePath, []byte("mock archive"), asdf.CommonFilePermission)).To(Succeed())

				gcloudExtractTarGz = func(string, string) error {
					return errForcedGcloudExtractError
				}

				err = plugin.Install(context.Background(), version, downloadDir, installDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("extracting archive"))
			})

			It("returns unsupported platform error when runtime OS is not supported", func() {
				version := "548.0.0"
				plugin := newPluginWithRuntime(fmt.Sprintf(gcsAPIURL, gcsBucketName), gcloudRuntime{goos: "windows", arch: "amd64"})

				downloadDir, err := os.MkdirTemp("", "gcloud-install-download-unsupportedos-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "gcloud-install-target-unsupportedos-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				err = plugin.Install(context.Background(), version, downloadDir, installDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported platform"))
			})

			It("returns error when ensureToolchains fails", func() {
				version := "548.0.0"
				plugin := New()

				ensureToolchains = func(_ context.Context, _ ...string) error {
					return errForcedGcloudToolchainError
				}

				downloadDir, err := os.MkdirTemp("", "gcloud-install-download-toolchainfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "gcloud-install-target-toolchainfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				objectName, err := plugin.getObjectName(version)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, objectName)
				Expect(os.WriteFile(archivePath, []byte("mock archive"), asdf.CommonFilePermission)).To(Succeed())

				err = plugin.Install(context.Background(), version, downloadDir, installDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("forced toolchain error"))
			})

			It("returns error when installPythonToolchain fails", func() {
				version := "548.0.0"
				plugin := New()

				installPythonToolchain = func(_ context.Context) error {
					return errForcedGcloudPythonInstallError
				}

				downloadDir, err := os.MkdirTemp("", "gcloud-install-download-pythonfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "gcloud-install-target-pythonfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				objectName, err := plugin.getObjectName(version)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, objectName)
				Expect(os.WriteFile(archivePath, []byte("mock archive"), asdf.CommonFilePermission)).To(Succeed())

				err = plugin.Install(context.Background(), version, downloadDir, installDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("forced python install error"))
			})
		})
	})

	Describe("gcloudTestFixture helpers", func() {
		It("SetupVersions handles nil and non-nil version sets", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()

			fixture.SetupVersions([]string{"548.0.0", "549.0.0"})
			fixture.SetupVersions(nil)
		})

		It("SetupTagsFromGoldie returns error when goldie files are missing", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()

			originalPath := fixture.testdataPath
			fixture.testdataPath = filepath.Join(originalPath, "nonexistent-subdir")
			defer func() { fixture.testdataPath = originalPath }()

			err := fixture.SetupTagsFromGoldie()
			Expect(err).To(HaveOccurred())
		})

		It("GoldieFilterPattern returns error when versions cannot be read", func() {
			fixture := newGcloudTestFixture()
			defer fixture.Close()

			originalPath := fixture.testdataPath
			fixture.testdataPath = filepath.Join(originalPath, "nonexistent-subdir")
			defer func() { fixture.testdataPath = originalPath }()

			pattern, err := fixture.GoldieFilterPattern()
			Expect(err).To(HaveOccurred())
			Expect(pattern).To(BeEmpty())
		})
	})
})
