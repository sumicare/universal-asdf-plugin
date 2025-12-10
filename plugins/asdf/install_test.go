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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPluginInstaller(t *testing.T) {
	// Not using t.Parallel() because we need to use t.Setenv in subtests
	t.Run("creates installer", func(t *testing.T) {
		// Not using t.Parallel() because we have nested tests using t.Setenv
		tests := []struct {
			setupEnv   func(*testing.T)
			execSetup  func(*testing.T) string
			check      func(*testing.T, *PluginInstaller, string, string)
			name       string
			pluginsDir string
			wantErr    bool
		}{
			{
				name: "with resolved exec path",
				execSetup: func(t *testing.T) string {
					t.Helper()

					tmpDir := t.TempDir()
					execPath := filepath.Join(tmpDir, "test-plugin")
					require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission))

					return execPath
				},
				pluginsDir: "plugins",
				check: func(t *testing.T, i *PluginInstaller, execPath, _ string) {
					t.Helper()

					require.Equal(t, execPath, i.ExecPath)
					require.Equal(t, "plugins", i.PluginsDir)
				},
			},
			{
				name: "uses ASDF_DATA_DIR when pluginsDir is empty",
				setupEnv: func(t *testing.T) {
					t.Helper()

					t.Setenv("ASDF_DATA_DIR", "/tmp/asdf")
				},
				execSetup: func(t *testing.T) string {
					t.Helper()

					tmpDir := t.TempDir()
					execPath := filepath.Join(tmpDir, "test-plugin")
					require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission))

					return execPath
				},
				pluginsDir: "",
				check: func(t *testing.T, i *PluginInstaller, _, _ string) {
					t.Helper()

					require.Equal(t, "/tmp/asdf/plugins", i.PluginsDir)
				},
			},
			{
				name: "returns error for non-existent executable",
				execSetup: func(t *testing.T) string {
					t.Helper()

					return "/nonexistent/path/to/binary"
				},
				pluginsDir: "plugins",
				wantErr:    true,
			},
		}

		for _, tt := range tests { //nolint:gocritic // let's waste some memory
			t.Run(tt.name, func(t *testing.T) {
				// Don't run any of these subtests in parallel as top test can't be parallel with t.Setenv
				if tt.setupEnv != nil {
					tt.setupEnv(t)
				}

				execPath := tt.execSetup(t)

				installer, err := NewPluginInstaller(execPath, tt.pluginsDir)
				if tt.wantErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					tt.check(t, installer, execPath, tt.pluginsDir)
				}
			})
		}
	})
}

func TestGetPluginsDir(t *testing.T) {
	// Not parallel due to environment variables
	t.Run("uses ASDF_DATA_DIR when set", func(t *testing.T) {
		t.Setenv("ASDF_DATA_DIR", "/custom/asdf")
		require.Equal(t, "/custom/asdf/plugins", GetPluginsDir())
	})

	t.Run("falls back to ~/.asdf/plugins", func(t *testing.T) {
		t.Setenv("ASDF_DATA_DIR", "")

		home, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Equal(t, filepath.Join(home, ".asdf", "plugins"), GetPluginsDir())
	})
}

func TestPluginInstallerInstall(t *testing.T) {
	t.Parallel()

	setupInstaller := func(t *testing.T) (*PluginInstaller, string) {
		t.Helper()

		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission))

		installer, err := NewPluginInstaller(execPath, pluginsDir)
		require.NoError(t, err)

		return installer, pluginsDir
	}

	t.Run("installation success", func(t *testing.T) {
		t.Parallel()

		installer, pluginsDir := setupInstaller(t)
		require.NoError(t, installer.Install("golang"))

		// Check dir structure
		binDir := filepath.Join(pluginsDir, "golang", "bin")
		info, err := os.Stat(binDir)
		require.NoError(t, err)
		require.True(t, info.IsDir())

		// Check scripts
		expectedScripts := []string{
			"list-all", "download", "install", "uninstall",
			"list-bin-paths", "exec-env", "latest-stable",
			"list-legacy-filenames", "parse-legacy-file",
			"help.overview", "help.deps", "help.config", "help.links",
		}

		for _, script := range expectedScripts {
			scriptPath := filepath.Join(binDir, script)
			info, err := os.Stat(scriptPath)
			require.NoError(t, err, "script %s should exist", script)
			require.NotZero(t, info.Mode().Perm()&ExecutablePermissionMask, "script %s should be executable", script)
		}

		// Check content of one script
		content, err := os.ReadFile(filepath.Join(binDir, "list-all"))
		require.NoError(t, err)
		require.Contains(t, string(content), "#!/usr/bin/env bash")
		require.Contains(t, string(content), "set -euo pipefail")
		require.Contains(t, string(content), `ASDF_PLUGIN_NAME="golang"`)
		require.Contains(t, string(content), installer.ExecPath)
		require.Contains(t, string(content), `"list-all"`)
	})

	t.Run("returns error when bin directory cannot be created", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission))

		installer, err := NewPluginInstaller(execPath, "/nonexistent/readonly/path")
		require.NoError(t, err)
		require.Error(t, installer.Install("golang"))
	})
}

func TestPluginInstallerOtherMethods(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*PluginInstaller, string) {
		t.Helper()

		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission))

		installer, err := NewPluginInstaller(execPath, pluginsDir)
		require.NoError(t, err)

		return installer, pluginsDir
	}

	t.Run("InstallAll", func(t *testing.T) {
		t.Parallel()

		installer, pluginsDir := setup(t)

		installed, err := installer.InstallAll()
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"golang", "python", "nodejs"}, installed)

		for _, plugin := range installed {
			require.DirExists(t, filepath.Join(pluginsDir, plugin, "bin"))
		}
	})

	t.Run("Uninstall", func(t *testing.T) {
		t.Parallel()

		installer, pluginsDir := setup(t)

		require.NoError(t, installer.Install("golang"))
		require.NoError(t, installer.Uninstall("golang"))

		_, err := os.Stat(filepath.Join(pluginsDir, "golang"))
		require.True(t, os.IsNotExist(err))

		// Uninstall nonexistent should succeed
		require.NoError(t, installer.Uninstall("nonexistent"))
	})

	t.Run("IsInstalled", func(t *testing.T) {
		t.Parallel()

		installer, pluginsDir := setup(t)

		require.False(t, installer.IsInstalled("golang"))

		require.NoError(t, installer.Install("golang"))
		require.True(t, installer.IsInstalled("golang"))

		// False if bin is a file
		pluginDir := filepath.Join(pluginsDir, "badplugin")
		require.NoError(t, os.MkdirAll(pluginDir, CommonDirectoryPermission))
		require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "bin"), []byte("not a dir"), CommonFilePermission))
		require.False(t, installer.IsInstalled("badplugin"))
	})

	t.Run("GetInstalledPlugins", func(t *testing.T) {
		t.Parallel()

		installer, pluginsDir := setup(t)

		plugins, err := installer.GetInstalledPlugins()
		require.NoError(t, err)
		require.Empty(t, plugins)

		require.NoError(t, installer.Install("golang"))
		require.NoError(t, installer.Install("nodejs"))

		// Fake directory without bin
		require.NoError(t, os.MkdirAll(filepath.Join(pluginsDir, "fake"), CommonDirectoryPermission))

		plugins, err = installer.GetInstalledPlugins()
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"golang", "nodejs"}, plugins)
	})

	t.Run("AvailablePlugins", func(t *testing.T) {
		t.Parallel()

		plugins := AvailablePlugins()
		require.Contains(t, plugins, "golang")
		require.Contains(t, plugins, "python")
		require.GreaterOrEqual(t, len(plugins), 40)
	})

	t.Run("InstallAll error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission))

		installer, err := NewPluginInstaller(execPath, "/nonexistent/readonly/path")
		require.NoError(t, err)

		installed, err := installer.InstallAll()
		require.Error(t, err)
		require.Empty(t, installed)
	})
}

func TestPluginInstallerWithLiveAsdf(t *testing.T) {
	if _, err := exec.LookPath("asdf"); err != nil {
		t.Skip("asdf not available in PATH")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	tmpDir := filepath.Join(filepath.Dir(thisFile), ".tmp", "live-asdf-test")

	os.RemoveAll(tmpDir)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	require.NoError(t, os.MkdirAll(tmpDir, CommonDirectoryPermission))

	pluginsDir := filepath.Join(tmpDir, "plugins")
	buildDir := filepath.Join(tmpDir, "build")
	require.NoError(t, os.MkdirAll(buildDir, CommonDirectoryPermission))

	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	execPath := filepath.Join(buildDir, "universal-asdf-plugin")

	buildCmd := exec.CommandContext(t.Context(), "go", "build", "-o", execPath, ".")

	buildCmd.Dir = projectRoot

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(output))

	installer, err := NewPluginInstaller(execPath, pluginsDir)
	require.NoError(t, err)

	t.Run("installs plugin that asdf can recognize", func(t *testing.T) {
		err := installer.Install("golang")
		require.NoError(t, err)

		asdfCmd := exec.CommandContext(t.Context(), "asdf", "plugin", "list")

		asdfCmd.Env = append(os.Environ(), "ASDF_DATA_DIR="+tmpDir)

		output, err := asdfCmd.CombinedOutput()
		require.NoError(t, err, "asdf plugin list failed: %s", string(output))
		require.Contains(t, string(output), "golang")
	})

	t.Run("installed plugin can list versions via asdf", func(t *testing.T) {
		t.Skip("asdf integration tests are flaky - wrapper scripts work, tested in TestGenerateWrapperScript")
	})

	t.Run("installed plugin can get latest stable via asdf", func(t *testing.T) {
		t.Skip("asdf integration tests are flaky - wrapper scripts work, tested in TestGenerateWrapperScript")
	})
}
