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

package asdf_plugin_argo

import (
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

var (
	// errForcedHTTPError simulates a failing HTTP client for Argo tests.
	errForcedHTTPError = errors.New("forced HTTP error")
	// errEnsureToolchainsFailed simulates a toolchain ensure failure in Argo tests.
	errEnsureToolchainsFailed = errors.New("ensure toolchains failed")
	// errDownloadFailed simulates a download failure in Argo tests.
	errDownloadFailed = errors.New("download failed")
	// errEnsureToolVersionsFailed simulates a tool-versions ensure failure in Argo tests.
	errEnsureToolVersionsFailed = errors.New("ensure tool-versions failed")
)

// testdataPath returns the path to this plugin's testdata directory.
func testdataPath() string {
	_, file, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(file), "testdata")
}

// pluginTestConfig returns the test configuration for the argo plugin.
// It provides metadata and constructors for use with shared goldie test helpers.
func pluginTestConfig() *testutil.PluginTestConfig {
	return &testutil.PluginTestConfig{
		Config: &asdf.BinaryPluginConfig{
			Name:       "argo",
			RepoOwner:  "argoproj",
			RepoName:   "argo-workflows",
			BinaryName: "argo",

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

var _ = Describe("Argo Plugin", func() {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := New()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal("argo"))
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
			Expect(plugin.Name()).To(Equal("argo"))
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
			tempDir, err := os.MkdirTemp("", "argo-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			versionFile := filepath.Join(tempDir, ".argo-version")
			Expect(os.WriteFile(versionFile, []byte("3.7.5\n"), asdf.CommonFilePermission)).To(Succeed())

			version, err := plugin.ParseLegacyFile(versionFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("3.7.5"))
		})

		It("returns error when file cannot be read", func() {
			plugin := New()
			_, err := plugin.ParseLegacyFile(filepath.Join(os.TempDir(), "nonexistent-argo-version-file"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := New()
			tempDir, err := os.MkdirTemp("", "argo-uninstall-*")
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
			Expect(help.Overview).To(ContainSubstring("Argo"))
			Expect(help.Deps).To(ContainSubstring("Go"))
			Expect(help.Config).NotTo(BeEmpty())
			Expect(help.Links).To(ContainSubstring("github.com"))
		})
	})

	Describe("ListAll", func() {
		It("lists Argo versions from goldie snapshots", func() {
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
		})

		It("lists Argo versions from GitHub", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Argo GitHub ListAll test")
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
	})

	Describe("LatestStable", func() {
		It("returns latest stable version from goldie snapshots", func() {
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

			goldieLatest, gErr := fixture.GoldieLatest()
			Expect(gErr).NotTo(HaveOccurred())
			Expect(version).To(Equal(goldieLatest))
		})

		It("returns latest stable version", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Argo GitHub LatestStable test")
			}

			plugin := New()
			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())
			Expect(version).To(HavePrefix("3."))
		})

		It("filters by pattern derived from latest version", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Argo GitHub LatestStable pattern test")
			}

			plugin := New()
			latest, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HavePrefix("3."))

			prefix := latest
			if len(latest) > 3 {
				prefix = latest[:3]
			}

			version, err := plugin.LatestStable(context.Background(), prefix)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(HavePrefix(prefix))
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
		It("is a no-op for argo (install handles source download)", func() {
			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixture(cfg, 3)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			version, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			tempDir, err := os.MkdirTemp("", "argo-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = fixture.Plugin.Download(context.Background(), version, tempDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Install", func() {
		It("installs argo using a fake go binary via go build", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Argo Install test")
			}

			cfg := pluginTestConfig()
			fixture := testutil.NewBinaryPluginTestFixture(cfg, 3)
			defer fixture.Close()

			originalInstallBuildToolchainsFunc := installBuildToolchainsFunc
			installBuildToolchainsFunc = func(context.Context) error { return nil }
			defer func() {
				installBuildToolchainsFunc = originalInstallBuildToolchainsFunc
			}()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with -update (and ONLINE=1) to create")
			}

			version, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			installDir, err := os.MkdirTemp("", "argo-install-*")
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
				"if [ -z \"$out\" ]; then out=\"${GOBIN:-.}/argo\"; fi\n" +
				"mkdir -p \"$(dirname \"$out\")\"\n" +
				"echo '#!/bin/sh' > \"$out\"\n" +
				"echo 'echo argo' >> \"$out\"\n" +
				"chmod +x \"$out\"\n")
			Expect(os.WriteFile(fakeGoPath, script, asdf.CommonDirectoryPermission)).To(Succeed())

			originalPath := os.Getenv("PATH")
			defer os.Setenv("PATH", originalPath)
			Expect(os.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+originalPath)).To(Succeed())

			plugin := New()

			err = plugin.Install(context.Background(), version, "", installDir)
			Expect(err).NotTo(HaveOccurred())

			binDir := filepath.Join(installDir, "bin")
			binaryPath := filepath.Join(binDir, "argo")
			info, statErr := os.Stat(binaryPath)
			Expect(statErr).NotTo(HaveOccurred())
			Expect(info.Mode()&asdf.ExecutablePermissionMask).NotTo(BeZero(), "argo binary should be executable")
		})

		It("returns error when ensuring toolchains fails", func() {
			if !asdf.IsOnline() {
				Skip("ONLINE=1 required for Argo Install error path test")
			}

			originalEnsure := ensureToolchainsFn
			ensureToolchainsFn = func(context.Context, ...string) error {
				return errEnsureToolchainsFailed
			}
			defer func() { ensureToolchainsFn = originalEnsure }()

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-toolchain-fail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("ensure toolchains failed"))
		})

		It("returns error when go is not found in PATH", func() {
			installDir, err := os.MkdirTemp("", "argo-install-nogo-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			originalPath := os.Getenv("PATH")
			defer os.Setenv("PATH", originalPath)
			Expect(os.Setenv("PATH", "")).To(Succeed())

			plugin := New()
			_, err = exec.LookPath("go")
			Expect(err).To(HaveOccurred())

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("go is required to install argo"))
		})

		It("returns error when downloading argo source fails", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(context.Context, string, string) error {
				return errDownloadFailed
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-downloadfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("downloading argo source"))
		})

		It("returns error when tar extraction fails", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(_ context.Context, _, _ string) error {
				return nil
			}

			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if name == "tar" {
					return exec.CommandContext(ctx, "false")
				}

				return originalExecFn(ctx, name, args...)
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-tarfail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("extracting argo source"))
		})

		It("returns error when tool-versions file cannot be ensured", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(_ context.Context, _, _ string) error {
				return nil
			}

			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if name == "tar" {
					return exec.CommandContext(ctx, "true")
				}

				return originalExecFn(ctx, name, args...)
			}

			originalEnsureToolVersions := ensureToolVersionsFileFn
			defer func() { ensureToolVersionsFileFn = originalEnsureToolVersions }()

			ensureToolVersionsFileFn = func(context.Context, string, ...string) error {
				return errEnsureToolVersionsFailed
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-toolversions-fail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("ensure tool-versions failed"))
		})

		It("returns error when installing UI dependencies fails", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(_ context.Context, _, _ string) error {
				return nil
			}

			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if len(args) >= 3 && args[0] == "exec" && args[1] == "yarn" && args[2] == "install" {
					return exec.CommandContext(ctx, "false")
				}

				if name == "tar" || (len(args) >= 3 && args[0] == "exec" && args[1] == "yarn" && args[2] == "build") {
					return exec.CommandContext(ctx, "true")
				}

				return originalExecFn(ctx, name, args...)
			}

			originalEnsureToolVersions := ensureToolVersionsFileFn
			defer func() { ensureToolVersionsFileFn = originalEnsureToolVersions }()

			ensureToolVersionsFileFn = func(context.Context, string, ...string) error {
				return nil
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-yarn-install-fail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
			Expect(installErr.Error()).To(ContainSubstring("installing argo UI dependencies with yarn"))
		})

		It("returns error when building UI fails", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(_ context.Context, _, _ string) error {
				return nil
			}

			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if name == "tar" || (len(args) >= 3 && args[0] == "exec" && args[1] == "yarn" && args[2] == "install") {
					return exec.CommandContext(ctx, "true")
				}

				if len(args) >= 3 && args[0] == "exec" && args[1] == "yarn" && args[2] == "build" {
					return exec.CommandContext(ctx, "false")
				}

				return originalExecFn(ctx, name, args...)
			}

			originalEnsureToolVersions := ensureToolVersionsFileFn
			defer func() { ensureToolVersionsFileFn = originalEnsureToolVersions }()

			ensureToolVersionsFileFn = func(context.Context, string, ...string) error {
				return nil
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-yarn-build-fail-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
		})

		It("returns error when argo binary is missing after install", func() {
			originalDownloadFileFn := downloadFileFn
			defer func() { downloadFileFn = originalDownloadFileFn }()

			downloadFileFn = func(_ context.Context, _, _ string) error {
				return nil
			}

			originalExecFn := execCommandContextFn
			defer func() { execCommandContextFn = originalExecFn }()

			execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if name == "tar" || name == "go" || (len(args) >= 3 && args[0] == "exec" && args[1] == "yarn") {
					return exec.CommandContext(ctx, "true")
				}

				return originalExecFn(ctx, name, args...)
			}

			originalEnsureToolVersions := ensureToolVersionsFileFn
			defer func() { ensureToolVersionsFileFn = originalEnsureToolVersions }()

			ensureToolVersionsFileFn = func(context.Context, string, ...string) error {
				return nil
			}

			originalStatFn := statFn
			defer func() { statFn = originalStatFn }()

			statFn = func(string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			}

			plugin := New()
			installDir, err := os.MkdirTemp("", "argo-install-nobinary-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			installErr := plugin.Install(context.Background(), "3.7.5", "", installDir)
			Expect(installErr).To(HaveOccurred())
		})
	})
})
