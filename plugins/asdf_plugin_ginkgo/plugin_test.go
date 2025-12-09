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

package asdf_plugin_ginkgo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errForcedHTTPError simulates a failing HTTP client for ginkgo tests.
	errForcedHTTPError = errors.New("forced HTTP error")
	// errBinMkdirFailed simulates a bin directory creation failure in ginkgo tests.
	errBinMkdirFailed = errors.New("bin mkdir failed")
	// errDownloadFailed simulates a source download failure in ginkgo tests.
	errDownloadFailed = errors.New("download failed")
)

// testdataPath returns the path to this plugin's testdata directory.
func testdataPath() string {
	_, file, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(file), "testdata")
}

// pluginTestConfig returns the test configuration for the ginkgo plugin.
func pluginTestConfig() *testutil.PluginTestConfig {
	return &testutil.PluginTestConfig{
		Config: &asdf.BinaryPluginConfig{
			Name:       "ginkgo",
			RepoOwner:  "onsi",
			RepoName:   "ginkgo",
			BinaryName: "ginkgo",

			VersionPrefix: "v",
		},
		TestdataPath:        testdataPath(),
		NewPlugin:           func() asdf.Plugin { return New() },
		NewPluginWithClient: func(c *github.Client) asdf.Plugin { return NewWithClient(c) },
	}
}

// failingHTTPClient is an http.Client-like type that always returns an error.
type failingHTTPClient struct{}

// Do implements the minimal http.Client interface and always fails for tests.
func (*failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, errForcedHTTPError
}

var _ = Describe("Ginkgo Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("ginkgo"))
		})
	})

	Describe("NewWithClient", func() {
		It("creates a plugin with custom client", func() {
			client := github.NewClient()
			plugin := NewWithClient(client)
			Expect(plugin).NotTo(BeNil())
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := New()
			Expect(plugin.Name()).To(Equal("ginkgo"))
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
			tempDir, err := os.MkdirTemp("", "ginkgo-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			versionFile := filepath.Join(tempDir, ".ginkgo-version")
			err = os.WriteFile(versionFile, []byte("2.20.0\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			version, err := plugin.ParseLegacyFile(versionFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("2.20.0"))
		})

		It("returns error when file cannot be read", func() {
			plugin := New()
			_, err := plugin.ParseLegacyFile(filepath.Join(os.TempDir(), "nonexistent-ginkgo-version-file"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "ginkgo-uninstall-*")
			Expect(err).NotTo(HaveOccurred())

			installPath := filepath.Join(tempDir, "install")
			binDir := filepath.Join(installPath, "bin")
			Expect(os.MkdirAll(binDir, asdf.CommonDirectoryPermission)).To(Succeed())

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
			Expect(help.Overview).To(ContainSubstring("Ginkgo"))
			Expect(help.Deps).To(ContainSubstring("Go"))
			Expect(help.Config).NotTo(BeEmpty())
			Expect(help.Links).To(ContainSubstring("github.com"))
		})
	})

	Describe("ListAll", func() {
		It("lists Ginkgo v2 versions", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixtureWithMode(cfg, true)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			versions, err := fixture.Plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			goldieVersions, gErr := fixture.GoldieVersions()
			Expect(gErr).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements(goldieVersions))
			for _, v := range versions {
				Expect(v).To(HavePrefix("2."))
			}
		})

		It("propagates errors from the GitHub client", func() {
			failingClient := github.NewClientWithHTTP(&failingHTTPClient{}, "http://invalid-api")
			plugin := NewWithClient(failingClient)

			versions, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(versions).To(BeEmpty())
		})
	})

	Describe("LatestStable", func() {
		It("returns latest stable v2 version", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixtureWithMode(cfg, true)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.Plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
			Expect(version).To(HavePrefix("2."))

			goldieLatest, gErr := fixture.GoldieLatest()
			Expect(gErr).NotTo(HaveOccurred())
			Expect(version).To(Equal(goldieLatest))
		})

		It("filters by pattern", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixtureWithMode(cfg, true)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			filterPattern, err := fixture.GoldieFilterPattern()
			Expect(err).NotTo(HaveOccurred())

			version, err := fixture.Plugin.LatestStable(context.Background(), filterPattern)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(HavePrefix("2."))
			Expect(version).To(HavePrefix(filterPattern))
		})

		It("returns error when no versions match pattern", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixtureWithMode(cfg, true)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			_, err := fixture.Plugin.LatestStable(context.Background(), "99.99")
			Expect(err).To(HaveOccurred())
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
		})
	})

	Describe("Download", func() {
		It("is a no-op for ginkgo (install handles source download)", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixture(cfg, 3)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			version, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			tempDir, err := os.MkdirTemp("", "ginkgo-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = fixture.Plugin.Download(context.Background(), version, tempDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Install", func() {
		It("installs ginkgo using a fake go binary via go build", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixture(cfg, 3)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			version, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			installDir, err := os.MkdirTemp("", "ginkgo-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			fakeBinDir, err := os.MkdirTemp("", "fake-go-bin-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(fakeBinDir)

			fakeGoPath := filepath.Join(fakeBinDir, "go")
			script := []byte("#!/bin/sh\n" +
				"out=\"\"\n" +
				"prev=\"\"\n" +
				"for arg in \"$@\"; do \n" +
				"  if [ \"$prev\" = \"-o\" ]; then out=\"$arg\"; fi\n" +
				"  prev=\"$arg\"\n" +
				"done\n" +
				"if [ -z \"$out\" ]; then out=\"${GOBIN:-.}/ginkgo\"; fi\n" +
				"mkdir -p \"$(dirname \"$out\")\"\n" +
				"echo '#!/bin/sh' > \"$out\"\n" +
				"echo 'echo ginkgo' >> \"$out\"\n" +
				"chmod +x \"$out\"\n")
			Expect(os.WriteFile(fakeGoPath, script, asdf.CommonDirectoryPermission)).To(Succeed())

			originalPath := os.Getenv("PATH")
			defer os.Setenv("PATH", originalPath)
			Expect(os.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+originalPath)).To(Succeed())

			plugin := New()

			err = plugin.Install(context.Background(), version, "", installDir)
			Expect(err).NotTo(HaveOccurred())

			binDir := filepath.Join(installDir, "bin")
			binaryPath := filepath.Join(binDir, "ginkgo")
			info, statErr := os.Stat(binaryPath)
			Expect(statErr).NotTo(HaveOccurred())
			Expect(info.Mode()&asdf.ExecutablePermissionMask).NotTo(BeZero(), "ginkgo binary should be executable")
		})

		It("returns error when bin directory cannot be created", func() {
			originalMkdirAllFn := mkdirAllFn
			defer func() { mkdirAllFn = originalMkdirAllFn }()

			mkdirAllFn = func(path string, perm os.FileMode) error {
				if strings.HasSuffix(path, string(os.PathSeparator)+"bin") {
					return errBinMkdirFailed
				}

				return originalMkdirAllFn(path, perm)
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "ginkgo-install-mkdirfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "2.20.0", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("creating bin directory"))
		})

		It("returns error when downloading ginkgo source fails", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(context.Context, string, string) error {
				return errDownloadFailed
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "ginkgo-install-downloadfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "2.20.0", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("downloading ginkgo source"))
		})

		It("returns error when tar extraction fails", func() {
			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if name == "tar" {
					cmd := exec.CommandContext(ctx, "false")
					return cmd
				}

				return originalExecFn(ctx, name, args...)
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "ginkgo-install-tarfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "2.20.0", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("extracting ginkgo source"))
		})

		It("returns error when build fails", func() {
			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if len(args) > 0 && args[0] == "build" {
					cmd := exec.CommandContext(ctx, "false")
					return cmd
				}

				return originalExecFn(ctx, name, args...)
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "ginkgo-install-buildfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "2.20.0", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("building ginkgo"))
		})

		It("returns error when ginkgo binary is missing after install", func() {
			originalStatFn := statFn
			defer func() { statFn = originalStatFn }()

			statFn = func(_ string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "ginkgo-install-nobinary-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "2.20.0", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("ginkgo binary not found"))
		})

		It("returns error when go is not found in PATH", func() {
			installDir, err := os.MkdirTemp("", "ginkgo-install-nogo-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			originalPath := os.Getenv("PATH")
			defer os.Setenv("PATH", originalPath)
			Expect(os.Setenv("PATH", "")).To(Succeed())

			plugin := New()

			_, err = exec.LookPath("go")
			Expect(err).To(HaveOccurred())

			installErr := plugin.Install(context.Background(), "2.20.0", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("go is required to install ginkgo"))
		})
	})
})
