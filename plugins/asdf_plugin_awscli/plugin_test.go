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

package asdf_plugin_awscli

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// errForcedHTTPClientError simulates a failing HTTP client in awscli tests.
var errForcedHTTPClientError = errors.New("forced http client error")

// errForcedExtractError is a static error used to force extractZip failures in tests.
var errForcedExtractError = errors.New("forced extract error")

// roundTripperFunc adapts a function so it satisfies http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper by delegating to the underlying function.
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// failingHTTPClient is an http.Client-like type that always returns an error.
type failingHTTPClient struct{}

// Do implements the minimal http.Client interface and always fails for tests.
func (*failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, errForcedHTTPClientError
}

// writeMockAWSCLIZipResponse writes a minimal awscli zip archive to the response writer for tests.
func writeMockAWSCLIZipResponse(writer http.ResponseWriter) {
	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	file, errCreate := zipWriter.Create("awscli.txt")
	if errCreate != nil {
		http.Error(writer, errCreate.Error(), http.StatusInternalServerError)
		return
	}

	if _, errWrite := file.Write([]byte("mock awscli package")); errWrite != nil {
		http.Error(writer, errWrite.Error(), http.StatusInternalServerError)
		return
	}

	if errClose := zipWriter.Close(); errClose != nil {
		http.Error(writer, errClose.Error(), http.StatusInternalServerError)
		return
	}

	if _, errWrite := writer.Write(buf.Bytes()); errWrite != nil {
		http.Error(writer, errWrite.Error(), http.StatusInternalServerError)
	}
}

var _ = Describe("Awscli Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("awscli"))
		})

		It("uses cached download when file already exists", func() {
			plugin := New()

			version := "2.32.9"
			url, err := plugin.getDownloadURL(version)
			if err != nil {
				Skip("unsupported platform for awscli Download cached test")
			}

			filename := filepath.Base(url)
			downloadDir, err := os.MkdirTemp("", "awscli-download-cached-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			filePath := filepath.Join(downloadDir, filename)

			content := make([]byte, 0, 2048)
			for i := range content {
				content = append(content, byte(i%256))
			}
			Expect(os.WriteFile(filePath, content, asdf.CommonFilePermission)).To(Succeed())

			Expect(plugin.Download(context.Background(), version, downloadDir)).To(Succeed())
		})
	})

	Describe("NewWithClient", func() {
		It("creates a plugin with custom client", func() {
			fixture := newAwscliTestFixture()
			defer fixture.Close()
			Expect(fixture.plugin).NotTo(BeNil())
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := New()
			Expect(plugin.Name()).To(Equal("awscli"))
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

	Describe("ParseLegacyFile", func() {
		It("parses version from file", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "awscli-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			testdataPath := testutil.GoldieTestDataPath(GinkgoT())
			testVersion, err := testutil.ReadGoldieLatest(testdataPath, "awscli_latest_stable.golden")
			if err != nil {
				testVersion = "2.32.9"
			}

			versionFile := filepath.Join(tempDir, ".awscli-version")
			err = os.WriteFile(versionFile, []byte(testVersion+"\n"), testutil.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := plugin.ParseLegacyFile(versionFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(testVersion))
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "awscli-uninstall-*")
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
			Expect(help.Overview).To(ContainSubstring("AWS CLI"))
			Expect(help.Links).To(ContainSubstring("aws.amazon.com"))
		})
	})

	Describe("ListAll", func() {
		var fixture *awscliTestFixture

		BeforeEach(func() {
			fixture = newAwscliMockFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("lists AWS CLI versions from goldie snapshots", func() {
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

		It("lists AWS CLI versions from GitHub when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for awscli GitHub ListAll test")
			}

			plugin := New()
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
		})

		It("propagates errors from the GitHub client", func() {
			failingClient := github.NewClientWithHTTP(&failingHTTPClient{}, "http://invalid-api")
			plugin := NewWithClient(failingClient)

			versions, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(versions).To(BeEmpty())
		})

		It("filters out non-v2 versions (mock-only)", func() {
			fixture.Close()
			fixture = newAwscliMockFixture()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			versions, err := fixture.plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			for _, v := range versions {
				Expect(v).To(HavePrefix("2."))
			}
		})
	})

	Describe("LatestStable", func() {
		var fixture *awscliTestFixture

		BeforeEach(func() {
			fixture = newAwscliMockFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns latest stable version from goldie snapshots", func() {
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

		It("returns latest stable version from GitHub when ONLINE=1", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for awscli GitHub LatestStable test")
			}

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
			Expect(version).To(HavePrefix("2."))
		})

		It("filters by pattern", func() {
			fixture.Close()
			fixture = newAwscliMockFixture()

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
			fixture.Close()
			fixture = newAwscliMockFixture()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			_, err := fixture.plugin.LatestStable(context.Background(), "nonexistent-version-pattern")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions matching"))
		})

		It("returns error when no v2 versions found", func() {
			fixture.Close()
			fixture = newAwscliMockFixture()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			fixture.SetupTags(nil)
			fixture.SetupTags([]string{"1.32.0"})

			_, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions found"))
		})

		It("returns error when no versions are available", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := w.Write([]byte("[]")); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}))
			defer server.Close()

			client := github.NewClientWithHTTP(server.Client(), server.URL)
			plugin := NewWithClient(client)
			_, err := plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no versions found"))
		})
	})

	Describe("getDownloadURL", func() {
		It("returns a valid URL for the current platform", func() {
			plugin := New()

			url, err := plugin.getDownloadURL("2.32.9")
			Expect(err).NotTo(HaveOccurred())
			Expect(url).To(ContainSubstring("2.32.9"))
			Expect(url).To(ContainSubstring("awscli"))
		})

		It("returns linux amd64 download URL", func() {
			client := newAwscliMockFixture().plugin.githubClient
			plugin := newPluginWithRuntime(client, awscliRuntime{goos: "linux", arch: "amd64"})

			url, err := plugin.getDownloadURL("2.32.9")
			Expect(err).NotTo(HaveOccurred())
			Expect(url).To(ContainSubstring("awscli-exe-linux-x86_64-2.32.9.zip"))
		})

		It("returns linux arm64 download URL", func() {
			client := newAwscliMockFixture().plugin.githubClient
			plugin := newPluginWithRuntime(client, awscliRuntime{goos: "linux", arch: "arm64"})

			url, err := plugin.getDownloadURL("2.32.9")
			Expect(err).NotTo(HaveOccurred())
			Expect(url).To(ContainSubstring("awscli-exe-linux-aarch64-2.32.9.zip"))
		})

		It("returns darwin pkg URL", func() {
			client := newAwscliMockFixture().plugin.githubClient
			plugin := newPluginWithRuntime(client, awscliRuntime{goos: "darwin", arch: "amd64"})

			url, err := plugin.getDownloadURL("2.32.9")
			Expect(err).NotTo(HaveOccurred())
			Expect(url).To(ContainSubstring("AWSCLIV2-2.32.9.pkg"))
		})

		It("returns error for unsupported platform", func() {
			client := newAwscliMockFixture().plugin.githubClient
			plugin := newPluginWithRuntime(client, awscliRuntime{goos: "windows", arch: "amd64"})

			url, err := plugin.getDownloadURL("2.32.9")
			Expect(err).To(HaveOccurred())
			Expect(url).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("unsupported platform"))
		})
	})

	Describe("Download", func() {
		var fixture *awscliTestFixture
		var server *httptest.Server
		var originalHTTPClient *http.Client
		var originalBaseURL string

		BeforeEach(func() {
			fixture = newAwscliMockFixture()
			originalHTTPClient = awscliHTTPClient
			originalBaseURL = awscliDownloadBaseURL
		})

		AfterEach(func() {
			awscliHTTPClient = originalHTTPClient
			awscliDownloadBaseURL = originalBaseURL
			if server != nil {
				server.Close()
			}
			fixture.Close()
		})

		It("downloads AWS CLI package using a mocked HTTP client (always on)", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			tempDir, err := os.MkdirTemp("", "awscli-download-mock-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeMockAWSCLIZipResponse(w)
			}))

			awscliDownloadBaseURL = server.URL

			awscliHTTPClient = server.Client()

			err = fixture.plugin.Download(context.Background(), version, tempDir)
			Expect(err).NotTo(HaveOccurred())

			files, err := os.ReadDir(tempDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).ToNot(BeEmpty())
		})

		It("returns error when HTTP status is not OK", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			tempDir, err := os.MkdirTemp("", "awscli-download-status-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))

			awscliDownloadBaseURL = server.URL
			awscliHTTPClient = server.Client()

			err = fixture.plugin.Download(context.Background(), version, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("download failed"))
		})

		It("returns error when HTTP client fails", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			tempDir, err := os.MkdirTemp("", "awscli-download-http-error-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			awscliDownloadBaseURL = "https://awscli.invalid.test"
			awscliHTTPClient = &http.Client{
				Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, errForcedHTTPClientError
				}),
			}

			err = fixture.plugin.Download(context.Background(), version, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("downloading awscli"))
		})

		It("returns error when extracting zip fails", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			originalExtractFn := extractZipFn
			defer func() { extractZipFn = originalExtractFn }()

			extractZipFn = func(string, string) error { return errForcedExtractError }

			tempDir, err := os.MkdirTemp("", "awscli-download-extractfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeMockAWSCLIZipResponse(w)
			}))

			awscliDownloadBaseURL = server.URL
			awscliHTTPClient = server.Client()

			plugin := fixture.plugin
			plugin.runtime.goos = "linux"
			plugin.runtime.arch = "amd64"

			err = plugin.Download(context.Background(), version, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("extracting zip"))
		})

		It("returns error when platform is unsupported", func() {
			tempDir, err := os.MkdirTemp("", "awscli-download-unsupported-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			client := newAwscliMockFixture().plugin.githubClient
			plugin := newPluginWithRuntime(client, awscliRuntime{goos: "windows", arch: "amd64"})

			err = plugin.Download(context.Background(), "2.32.9", tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported platform"))
		})

		It("returns error when output file cannot be created", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}
			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())

			tempDir, err := os.MkdirTemp("", "awscli-download-openfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			Expect(os.Chmod(tempDir, 0o555)).To(Succeed())

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeMockAWSCLIZipResponse(w)
			}))

			awscliDownloadBaseURL = server.URL
			awscliHTTPClient = server.Client()

			plugin := fixture.plugin
			plugin.runtime.goos = "linux"
			plugin.runtime.arch = "amd64"

			err = plugin.Download(context.Background(), version, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating file"))
		})

		It("returns error when request cannot be created", func() {
			plugin := New()
			plugin.runtime.goos = "linux"
			plugin.runtime.arch = "amd64"

			savedBaseURL := awscliDownloadBaseURL
			defer func() { awscliDownloadBaseURL = savedBaseURL }()

			awscliDownloadBaseURL = "://bad-url"

			tempDir, err := os.MkdirTemp("", "awscli-download-badrequest-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "2.32.9", tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating request"))
		})
	})

	Describe("Install", func() {
		var (
			tempDir     string
			downloadDir string
			err         error
		)

		BeforeEach(func() {
			tempDir, err = os.MkdirTemp("", "awscli-install-*")
			Expect(err).NotTo(HaveOccurred())

			downloadDir = filepath.Join(tempDir, "download")
			Expect(os.MkdirAll(downloadDir, asdf.CommonDirectoryPermission)).To(Succeed())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		Context("on Linux", func() {
			BeforeEach(func() {
				if runtime.GOOS != "linux" {
					Skip("Linux-only test")
				}

				awsDir := filepath.Join(downloadDir, "aws")
				Expect(os.MkdirAll(awsDir, asdf.CommonDirectoryPermission)).To(Succeed())
				installerPath := filepath.Join(awsDir, "install")

				script := `#!/bin/sh
set -e
lib=""
bin=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -i)
      shift
      lib="$1"
      ;;
    -b)
      shift
      bin="$1"
      ;;
  esac
  shift
done

if [ -n "$lib" ]; then
  mkdir -p "$lib"
fi

if [ -n "$bin" ]; then
  mkdir -p "$bin"
  touch "$bin/aws"
fi
`
				Expect(os.WriteFile(installerPath, []byte(script), asdf.CommonFilePermission)).To(Succeed())
			})

			It("returns error when installer exits with non-zero status", func() {
				plugin := New()

				awsDir := filepath.Join(downloadDir, "aws")
				installerPath := filepath.Join(awsDir, "install")
				script := "#!/bin/sh\nexit 1\n"
				Expect(os.WriteFile(installerPath, []byte(script), asdf.CommonFilePermission)).To(Succeed())

				err = plugin.Install(context.Background(), "2.0.0", downloadDir, tempDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("running installer"))
			})

			It("installs AWS CLI", func() {
				plugin := New()
				err = plugin.Install(context.Background(), "2.0.0", downloadDir, tempDir)
				Expect(err).NotTo(HaveOccurred())

				_, err = os.Stat(filepath.Join(tempDir, "bin", "aws"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns error when installer script is missing", func() {
				plugin := New()

				missingDownloadDir := filepath.Join(tempDir, "missing-download")
				Expect(os.MkdirAll(missingDownloadDir, asdf.CommonDirectoryPermission)).To(Succeed())

				err = plugin.Install(context.Background(), "2.0.0", missingDownloadDir, tempDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("making installer executable"))
			})
		})

		Context("with darwin runtime (simulated)", func() {
			It("installs AWS CLI using stubbed commands", func() {
				installDir, err := os.MkdirTemp("", "awscli-install-darwin-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				downloadDir, err := os.MkdirTemp("", "awscli-download-darwin-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				pkgPath := filepath.Join(downloadDir, "AWSCLIV2-2.0.0.pkg")
				Expect(os.WriteFile(pkgPath, []byte("pkg"), asdf.CommonFilePermission)).To(Succeed())

				originalExecFn := execCommandContextFn
				defer func() { execCommandContextFn = originalExecFn }()

				execCommandContextFn = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
					return exec.CommandContext(ctx, "true")
				}

				plugin := newPluginWithRuntime(github.NewClient(), awscliRuntime{goos: "darwin", arch: "amd64"})
				err = plugin.Install(context.Background(), "2.0.0", downloadDir, installDir)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns error when pkg extraction fails (darwin simulated)", func() {
				installDir, err := os.MkdirTemp("", "awscli-install-darwin-tarfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				downloadDir, err := os.MkdirTemp("", "awscli-download-darwin-tarfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				pkgPath := filepath.Join(downloadDir, "AWSCLIV2-2.0.0.pkg")
				Expect(os.WriteFile(pkgPath, []byte("pkg"), asdf.CommonFilePermission)).To(Succeed())

				originalExecFn := execCommandContextFn
				defer func() { execCommandContextFn = originalExecFn }()

				execCommandContextFn = func(ctx context.Context, name string, _ ...string) *exec.Cmd {
					if name == "pkgutil" {
						return exec.CommandContext(ctx, "false")
					}

					return exec.CommandContext(ctx, "true")
				}

				plugin := newPluginWithRuntime(github.NewClient(), awscliRuntime{goos: "darwin", arch: "amd64"})
				err = plugin.Install(context.Background(), "2.0.0", downloadDir, installDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("extracting pkg"))
			})

			It("returns error when copying aws-cli fails (darwin simulated)", func() {
				installDir, err := os.MkdirTemp("", "awscli-install-darwin-cpfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				downloadDir, err := os.MkdirTemp("", "awscli-download-darwin-cpfail-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				pkgPath := filepath.Join(downloadDir, "AWSCLIV2-2.0.0.pkg")
				Expect(os.WriteFile(pkgPath, []byte("pkg"), asdf.CommonFilePermission)).To(Succeed())

				originalExecFn := execCommandContextFn
				defer func() { execCommandContextFn = originalExecFn }()

				execCommandContextFn = func(ctx context.Context, name string, _ ...string) *exec.Cmd {
					if name == "pkgutil" {
						return exec.CommandContext(ctx, "true")
					}

					if name == "cp" {
						return exec.CommandContext(ctx, "false")
					}

					return exec.CommandContext(ctx, "true")
				}

				plugin := newPluginWithRuntime(github.NewClient(), awscliRuntime{goos: "darwin", arch: "amd64"})
				err = plugin.Install(context.Background(), "2.0.0", downloadDir, installDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("copying aws-cli"))
			})
		})

		Context("on macOS", func() {
			BeforeEach(func() {
				if runtime.GOOS != "darwin" {
					Skip("macOS-only test")
				}

				pkgPath := filepath.Join(downloadDir, "AWSCLIV2-2.0.0.pkg")
				Expect(os.WriteFile(pkgPath, []byte("mock pkg"), asdf.CommonFilePermission)).To(Succeed())

				extractDir := filepath.Join(downloadDir, "extracted")
				Expect(os.MkdirAll(extractDir, asdf.CommonDirectoryPermission)).To(Succeed())
				awsCliSrc := filepath.Join(extractDir, "aws-cli.pkg", "Payload", "aws-cli")
				Expect(os.MkdirAll(awsCliSrc, asdf.CommonDirectoryPermission)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(awsCliSrc, "aws"), []byte("mock aws"), asdf.CommonFilePermission)).To(Succeed())
			})

			It("installs AWS CLI", func() {
				plugin := New()
				err = plugin.Install(context.Background(), "2.0.0", downloadDir, tempDir)
				Expect(err).NotTo(HaveOccurred())

				_, err = os.Stat(filepath.Join(tempDir, "bin", "aws"))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("on unsupported OS", func() {
			It("returns an unsupported install platform error", func() {
				client := newAwscliMockFixture().plugin.githubClient
				plugin := newPluginWithRuntime(client, awscliRuntime{goos: "windows", arch: "amd64"})

				err = plugin.Install(context.Background(), "2.0.0", downloadDir, tempDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported install platform"))
			})
		})
	})

	Describe("awscliTestFixture helpers", func() {
		It("SetupTags handles nil and non-nil tag sets", func() {
			fixture := newAwscliMockFixture()
			defer fixture.Close()

			fixture.SetupTags([]string{"v2.0.0", "v2.1.0"})

			fixture.SetupTags(nil)
		})

		It("SetupTagsFromGoldie returns error when goldie files are missing", func() {
			fixture := newAwscliMockFixture()
			defer fixture.Close()

			originalPath := fixture.testdataPath
			fixture.testdataPath = filepath.Join(originalPath, "nonexistent-subdir")
			defer func() { fixture.testdataPath = originalPath }()

			err := fixture.SetupTagsFromGoldie()
			Expect(err).To(HaveOccurred())
		})

		It("GoldieFilterPattern returns error when versions cannot be read", func() {
			fixture := newAwscliMockFixture()
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
