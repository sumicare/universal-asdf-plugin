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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errTestVersionLookupFailed = errors.New("version lookup failed")
	errTestInstallFailed       = errors.New("install failed")
	errTestNotFound            = errors.New("not found")
	errTestWdError             = errors.New("wd error")
	errTestHomeError           = errors.New("home error")
)

func TestToolchains(t *testing.T) { //nolint:tparallel // Not using t.Parallel() because subtests use t.Setenv
	t.Run("returns nil when no tools are provided", func(t *testing.T) {
		// Not using t.Parallel() because parent test can't be parallel with t.Setenv in other subtests
		require.NoError(t, EnsureToolchains(t.Context()))
		require.NoError(t, EnsureToolVersionsFile(t.Context(), filepath.Join(t.TempDir(), ".tool-versions")))
	})

	t.Run("ensures .tool-versions entries for tools", func(t *testing.T) {
		// Not parallel due to mocking
		t.Run("defaults to latest if asdf missing", func(t *testing.T) {
			mockExec(t, func(string) (string, error) { return "", errTestNotFound })

			tempDir := t.TempDir()
			homeDir := filepath.Join(tempDir, "home")
			require.NoError(t, os.MkdirAll(homeDir, CommonDirectoryPermission))

			mockOS(t, "", homeDir)

			require.NoError(t, EnsureToolchains(t.Context(), "golang"))

			toolVersionsPath := filepath.Join(homeDir, ".tool-versions")
			data, err := os.ReadFile(toolVersionsPath)
			require.NoError(t, err)
			require.Contains(t, string(data), "golang latest")
		})

		t.Run("resolves actual version when asdf is present", func(t *testing.T) {
			// Use custom mock function to return the version directly
			origLookPath := execLookPath
			origExecCmd := execCommandContext
			t.Cleanup(func() {
				execLookPath = origLookPath
				execCommandContext = origExecCmd
			})

			// Mock asdf as available
			execLookPath = func(string) (string, error) {
				return "/usr/bin/asdf", nil
			}

			// Mock the command to return the expected version
			execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "echo", "1.21.5")
			}

			tempDir := t.TempDir()
			homeDir := filepath.Join(tempDir, "home")
			require.NoError(t, os.MkdirAll(homeDir, CommonDirectoryPermission))

			mockOS(t, "", homeDir)

			require.NoError(t, EnsureToolchains(t.Context(), "golang"))

			toolVersionsPath := filepath.Join(homeDir, ".tool-versions")
			data, err := os.ReadFile(toolVersionsPath)
			require.NoError(t, err)
			require.Contains(t, string(data), "golang 1.21.5")
		})

		t.Run("prefers .tool-versions in working directory", func(t *testing.T) {
			mockExec(t, func(string) (string, error) { return "", errTestNotFound })

			tempDir := t.TempDir()
			cwd := filepath.Join(tempDir, "cwd")
			home := filepath.Join(tempDir, "home")

			require.NoError(t, os.MkdirAll(cwd, CommonDirectoryPermission))
			require.NoError(t, os.MkdirAll(home, CommonDirectoryPermission))
			require.NoError(t, os.WriteFile(filepath.Join(cwd, ".tool-versions"), []byte(""), CommonFilePermission))

			mockOS(t, cwd, home)

			require.NoError(t, EnsureToolchains(t.Context(), "python"))

			cwdData, err := os.ReadFile(filepath.Join(cwd, ".tool-versions"))
			require.NoError(t, err)
			require.Contains(t, string(cwdData), "python latest")

			_, err = os.Stat(filepath.Join(home, ".tool-versions"))
			require.True(t, os.IsNotExist(err))
		})
	})

	t.Run("EnsureToolVersionsFile", func(t *testing.T) {
		// Not parallel due to mockExec potentially leaking if mixed with non-parallel
		t.Run("updates a specific .tool-versions file without installing", func(t *testing.T) {
			mockExec(t, func(string) (string, error) { return "", errTestNotFound })

			tempDir := t.TempDir()
			toolVersionsPath := filepath.Join(tempDir, ".tool-versions")

			require.NoError(t, EnsureToolVersionsFile(t.Context(), toolVersionsPath, "python"))

			data, err := os.ReadFile(toolVersionsPath)
			require.NoError(t, err)
			require.Contains(t, string(data), "python latest")
		})

		t.Run("does not duplicate existing tool entries", func(t *testing.T) {
			mockExec(t, func(string) (string, error) { return "", errTestNotFound })

			tempDir := t.TempDir()
			toolVersionsPath := filepath.Join(tempDir, ".tool-versions")
			require.NoError(t, os.WriteFile(toolVersionsPath, []byte("python latest\n"), CommonFilePermission))

			require.NoError(t, EnsureToolVersionsFile(t.Context(), toolVersionsPath, "python"))

			data, err := os.ReadFile(toolVersionsPath)
			require.NoError(t, err)
			require.Equal(t, "python latest\n", string(data))
		})
	})

	t.Run("error scenarios", func(t *testing.T) {
		t.Parallel()

		t.Run("ensureToolVersionLine cannot read file", func(t *testing.T) {
			t.Parallel()

			err := ensureToolVersionLine(filepath.Join(t.TempDir(), "missing"), "python", "latest")
			require.Error(t, err)
			require.Contains(t, err.Error(), "reading")
		})

		t.Run("ensureToolVersionLine cannot write file", func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			file := filepath.Join(tempDir, ".tool-versions")
			require.NoError(t, os.WriteFile(file, []byte(""), 0o400)) // Read-only

			err := ensureToolVersionLine(file, "python", "latest")
			require.Error(t, err)
			require.Contains(t, err.Error(), "updating")
		})
	})

	t.Run("resolveToolVersionsPath returns error when cannot create file", func(t *testing.T) {
		// Mock HOME to a read-only directory
		tempDir := t.TempDir()
		readOnlyHome := filepath.Join(tempDir, "ro-home")
		require.NoError(t, os.Mkdir(readOnlyHome, 0o500)) // Read-only directory

		// Make sure we are not in a dir with .tool-versions so it falls back to HOME
		emptyWd := filepath.Join(tempDir, "empty-wd")
		require.NoError(t, os.Mkdir(emptyWd, 0o755))

		mockOS(t, emptyWd, readOnlyHome)

		_, err := resolveToolVersionsPath()
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating")
	})
}

func TestResolveAsdfLatestVersion(t *testing.T) {
	// Tests using mocking or env vars should not be parallel with others using same mocks
	t.Run("returns latest when asdf is missing", func(t *testing.T) {
		setupAsdfLatestTest(t, "", false, false)

		version := resolveAsdfLatestVersion(t.Context(), "golang")
		require.Equal(t, "latest", version)
	})

	t.Run("returns latest when asdf fails", func(t *testing.T) {
		setupAsdfLatestTest(t, "", true, true)

		version := resolveAsdfLatestVersion(t.Context(), "golang")
		require.Equal(t, "latest", version)
	})

	t.Run("returns resolved version", func(t *testing.T) {
		setupAsdfLatestTest(t, "1.22.0", true, false)

		version := resolveAsdfLatestVersion(t.Context(), "golang")
		require.Equal(t, "1.22.0", version)
	})

	t.Run("returns latest when asdf returns empty output", func(t *testing.T) {
		setupAsdfLatestTest(t, "", true, false)

		version := resolveAsdfLatestVersion(t.Context(), "golang")
		require.Equal(t, "latest", version)
	})
}

// setupAsdfLatestTest sets up a test environment for resolveAsdfLatestVersion tests
// by mocking the execCommandContext and execLookPath functions.
func setupAsdfLatestTest(t *testing.T, output string, asdfAvailable, asdfFails bool) { //nolint:revive // we're fine with a flag parameter here
	t.Helper()

	// Original execCommandContext function
	orig := execCommandContext
	t.Cleanup(func() {
		execCommandContext = orig
	})

	// Mock the exec function with appropriate behavior
	if asdfFails {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		}
	} else {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", output)
		}
	}

	origLookPath := execLookPath
	t.Cleanup(func() {
		execLookPath = origLookPath
	})

	if asdfAvailable {
		execLookPath = func(string) (string, error) {
			return "/usr/bin/asdf", nil
		}
	} else {
		execLookPath = func(string) (string, error) {
			return "", errTestNotFound
		}
	}
}

type mockPlugin struct {
	latestError   error
	installError  error
	latestVersion string
	installCalled bool
}

func (*mockPlugin) Name() string                                  { return "mock" }
func (*mockPlugin) ListAll(_ context.Context) ([]string, error)   { return nil, nil }
func (*mockPlugin) Download(_ context.Context, _, _ string) error { return nil }
func (mockPlugin *mockPlugin) Install(_ context.Context, _, _, _ string) error {
	mockPlugin.installCalled = true
	return mockPlugin.installError
}
func (*mockPlugin) ListBinPaths() string                        { return "" }
func (*mockPlugin) ExecEnv(_ string) map[string]string          { return nil }
func (*mockPlugin) Uninstall(_ context.Context, _ string) error { return nil }
func (mockPlugin *mockPlugin) LatestStable(_ context.Context, _ string) (string, error) {
	return mockPlugin.latestVersion, mockPlugin.latestError
}

func (*mockPlugin) ResolveVersion(_ context.Context, version string) (string, error) {
	return version, nil
}
func (*mockPlugin) ListLegacyFilenames() []string            { return nil }
func (*mockPlugin) ParseLegacyFile(_ string) (string, error) { return "", nil }
func (*mockPlugin) Help() PluginHelp                         { return PluginHelp{} }

func TestInstallToolchain_Sequential(t *testing.T) {
	// Not parallel because it uses t.Setenv
	t.Run("installs toolchain successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("ASDF_DATA_DIR", tempDir)

		plugin := &mockPlugin{
			latestVersion: "1.2.3",
		}

		err := InstallToolchain(t.Context(), "test-tool", plugin)
		require.NoError(t, err)
		require.True(t, plugin.installCalled)

		installPath := filepath.Join(tempDir, "installs", "test-tool", "1.2.3")
		downloadPath := filepath.Join(tempDir, "downloads", "test-tool", "1.2.3")

		_, err = os.Stat(installPath)
		require.NoError(t, err)

		_, err = os.Stat(downloadPath)
		require.NoError(t, err)
	})

	t.Run("uses ASDF_DATA_DIR when set", func(t *testing.T) {
		customDir := filepath.Join(t.TempDir(), "custom-asdf")
		t.Setenv("ASDF_DATA_DIR", customDir)

		plugin := &mockPlugin{
			latestVersion: "2.0.0",
		}

		err := InstallToolchain(t.Context(), "test-tool", plugin)
		require.NoError(t, err)

		installPath := filepath.Join(customDir, "installs", "test-tool", "2.0.0")
		require.DirExists(t, installPath)
	})

	t.Run("falls back to HOME/.asdf when ASDF_DATA_DIR not set", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("ASDF_DATA_DIR", "")

		plugin := &mockPlugin{
			latestVersion: "3.0.0",
		}

		err := InstallToolchain(t.Context(), "test-tool", plugin)
		require.NoError(t, err)

		installPath := filepath.Join(homeDir, ".asdf", "installs", "test-tool", "3.0.0")
		require.DirExists(t, installPath)
	})

	t.Run("returns error when LatestStable fails", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("ASDF_DATA_DIR", tempDir)

		plugin := &mockPlugin{
			latestError: errTestVersionLookupFailed,
		}

		err := InstallToolchain(t.Context(), "test-tool", plugin)
		require.Error(t, err)
		require.Contains(t, err.Error(), "determining latest version")
		require.False(t, plugin.installCalled)
	})

	t.Run("returns error when Install fails", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("ASDF_DATA_DIR", tempDir)

		plugin := &mockPlugin{
			latestVersion: "1.0.0",
			installError:  errTestInstallFailed,
		}

		err := InstallToolchain(t.Context(), "test-tool", plugin)
		require.Error(t, err)
		require.Contains(t, err.Error(), "installing test-tool")
		require.True(t, plugin.installCalled)
	})

	t.Run("returns error when creating download directory fails", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("ASDF_DATA_DIR", tempDir)

		// Create a file where "downloads" directory should be
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "downloads"), []byte("file"), 0o600))

		plugin := &mockPlugin{latestVersion: "1.0.0"}
		err := InstallToolchain(t.Context(), "tool", plugin)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating download directory")
	})

	t.Run("returns error when creating install directory fails", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("ASDF_DATA_DIR", tempDir)

		// Create a file where "installs" directory should be
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "installs"), []byte("file"), 0o600))

		plugin := &mockPlugin{latestVersion: "1.0.0"}
		err := InstallToolchain(t.Context(), "tool", plugin)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating install directory")
	})
}

func TestEnsureGitRepo(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Run("clones repository when it does not exist", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")

		err := EnsureGitRepo(
			t.Context(),
			repoPath,
			"https://github.com/asdf-vm/asdf.git",
			"Cloning test repo...",
			"Clone successful",
		)
		require.NoError(t, err)
		require.DirExists(t, repoPath)
		require.DirExists(t, filepath.Join(repoPath, ".git"))
	})

	t.Run("updates existing repository", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")

		// First clone
		err := EnsureGitRepo(t.Context(), repoPath, "https://github.com/asdf-vm/asdf.git", "", "")
		require.NoError(t, err)

		// Then update
		err = EnsureGitRepo(t.Context(), repoPath, "https://github.com/asdf-vm/asdf.git", "", "")
		require.NoError(t, err)
	})

	t.Run("handles empty messages gracefully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")

		err := EnsureGitRepo(t.Context(), repoPath, "https://github.com/asdf-vm/asdf.git", "", "")
		require.NoError(t, err)
	})

	t.Run("returns error for invalid git URL", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")

		err := EnsureGitRepo(t.Context(), repoPath, "https://invalid-url-that-does-not-exist.local/repo.git", "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cloning")
	})

	t.Run("returns error when parent directory cannot be created", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		// Create a read-only directory where parent should be created
		roParent := filepath.Join(tempDir, "ro-parent")
		require.NoError(t, os.Mkdir(roParent, 0o500))

		repoPath := filepath.Join(roParent, "subdir", "repo")

		err := EnsureGitRepo(t.Context(), repoPath, "http://example.com/repo", "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating directory")
	})
}

func TestToolchainsExtraCoverage(t *testing.T) {
	// Not parallel due to global function mocking
	t.Run("resolveToolVersionsPath returns error when home dir check fails", func(t *testing.T) {
		mockExec(t, nil)
		// We need to override osGetwd manually to return error, as mockOS doesn't support error injection
		origGetwd := osGetwd

		origUserHomeDir := osUserHomeDir
		defer func() {
			osGetwd = origGetwd
			osUserHomeDir = origUserHomeDir
		}()

		osGetwd = func() (string, error) { return "", errTestWdError }
		osUserHomeDir = func() (string, error) { return "", errTestHomeError }

		_, err := resolveToolVersionsPath()
		require.Error(t, err)
		require.Contains(t, err.Error(), "determining home directory")
	})

	t.Run("InstallToolchain returns error when download path exists as file", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("ASDF_DATA_DIR", tempDir)

		// Create file blocking download path
		downloadPath := filepath.Join(tempDir, "downloads", "tool", "1.0.0")
		require.NoError(t, os.MkdirAll(filepath.Dir(downloadPath), 0o755))
		require.NoError(t, os.WriteFile(downloadPath, []byte("file"), 0o600))

		plugin := &mockPlugin{latestVersion: "1.0.0"}
		err := InstallToolchain(t.Context(), "tool", plugin)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating download directory")
	})
}
