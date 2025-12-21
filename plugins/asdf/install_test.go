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

package asdf_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

func TestNewPluginInstaller(t *testing.T) {
	t.Run("creates installer", func(t *testing.T) {
		tests := []struct {
			setupEnv   func(*testing.T)
			execSetup  func(*testing.T) string
			check      func(*testing.T, *asdf.PluginInstaller, string, string)
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
					require.NoError(
						t,
						os.WriteFile(
							execPath,
							[]byte("#!/bin/bash\necho test"),
							asdf.CommonDirectoryPermission,
						),
					)

					return execPath
				},
				pluginsDir: "plugins",
				check: func(t *testing.T, i *asdf.PluginInstaller, execPath, _ string) {
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
					require.NoError(
						t,
						os.WriteFile(
							execPath,
							[]byte("#!/bin/bash\necho test"),
							asdf.CommonDirectoryPermission,
						),
					)

					return execPath
				},
				pluginsDir: "",
				check: func(t *testing.T, i *asdf.PluginInstaller, _, _ string) {
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

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // does not work with SetEnv
				if tt.setupEnv != nil {
					tt.setupEnv(t)
				}

				execPath := tt.execSetup(t)

				installer, err := asdf.NewPluginInstaller(execPath, tt.pluginsDir)
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
	t.Run("uses ASDF_DATA_DIR when set", func(t *testing.T) {
		t.Setenv("ASDF_DATA_DIR", "/custom/asdf")
		require.Equal(t, "/custom/asdf/plugins", asdf.GetPluginsDir())
	})

	t.Run("falls back to ~/.asdf/plugins", func(t *testing.T) {
		t.Setenv("ASDF_DATA_DIR", "")

		home, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Equal(t, filepath.Join(home, ".asdf", "plugins"), asdf.GetPluginsDir())
	})
}

func TestPluginInstallerInstall(t *testing.T) {
	t.Parallel()

	setupInstaller := func(t *testing.T) (*asdf.PluginInstaller, string) {
		t.Helper()

		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(
			t,
			os.WriteFile(
				execPath,
				[]byte("#!/bin/bash\necho test"),
				asdf.CommonDirectoryPermission,
			),
		)

		installer, err := asdf.NewPluginInstaller(execPath, pluginsDir)
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
			require.NotZero(
				t,
				info.Mode().Perm()&asdf.ExecutablePermissionMask,
				"script %s should be executable",
				script,
			)
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
		require.NoError(
			t,
			os.WriteFile(
				execPath,
				[]byte("#!/bin/bash\necho test"),
				asdf.CommonDirectoryPermission,
			),
		)

		installer, err := asdf.NewPluginInstaller(execPath, "/nonexistent/readonly/path")
		require.NoError(t, err)
		require.Error(t, installer.Install("golang"))
	})
}

func TestPluginInstallerOtherMethods(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*asdf.PluginInstaller, string) {
		t.Helper()

		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(
			t,
			os.WriteFile(
				execPath,
				[]byte("#!/bin/bash\necho test"),
				asdf.CommonDirectoryPermission,
			),
		)

		installer, err := asdf.NewPluginInstaller(execPath, pluginsDir)
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
		require.NoError(t, os.MkdirAll(pluginDir, asdf.CommonDirectoryPermission))
		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(pluginDir, "bin"),
				[]byte("not a dir"),
				asdf.CommonFilePermission,
			),
		)
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
		require.NoError(
			t,
			os.MkdirAll(filepath.Join(pluginsDir, "fake"), asdf.CommonDirectoryPermission),
		)

		plugins, err = installer.GetInstalledPlugins()
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"golang", "nodejs"}, plugins)
	})

	t.Run("AvailablePlugins", func(t *testing.T) {
		t.Parallel()

		plugins := asdf.AvailablePlugins()
		require.Contains(t, plugins, "golang")
		require.Contains(t, plugins, "python")
		require.GreaterOrEqual(t, len(plugins), 40)
	})

	t.Run("InstallAll error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(
			t,
			os.WriteFile(
				execPath,
				[]byte("#!/bin/bash\necho test"),
				asdf.CommonDirectoryPermission,
			),
		)

		installer, err := asdf.NewPluginInstaller(execPath, "/nonexistent/readonly/path")
		require.NoError(t, err)

		installed, err := installer.InstallAll()
		require.Error(t, err)
		require.Empty(t, installed)
	})

	t.Run("Uninstall", func(t *testing.T) {
		t.Parallel()

		t.Run("removes plugin directory", func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			execPath := filepath.Join(tmpDir, "test-plugin")
			require.NoError(
				t,
				os.WriteFile(
					execPath,
					[]byte("#!/bin/bash\necho test"),
					asdf.CommonDirectoryPermission,
				),
			)

			pluginsDir := filepath.Join(tmpDir, "plugins")
			require.NoError(
				t,
				os.MkdirAll(
					filepath.Join(pluginsDir, "golang", "bin"),
					asdf.CommonDirectoryPermission,
				),
			)

			installer, err := asdf.NewPluginInstaller(execPath, pluginsDir)
			require.NoError(t, err)

			err = installer.Uninstall("golang")
			require.NoError(t, err)

			_, err = os.Stat(filepath.Join(pluginsDir, "golang"))
			require.True(t, os.IsNotExist(err))
		})

		t.Run("returns no error for non-existent plugin", func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			execPath := filepath.Join(tmpDir, "test-plugin")
			require.NoError(
				t,
				os.WriteFile(
					execPath,
					[]byte("#!/bin/bash\necho test"),
					asdf.CommonDirectoryPermission,
				),
			)

			installer, err := asdf.NewPluginInstaller(execPath, filepath.Join(tmpDir, "plugins"))
			require.NoError(t, err)

			err = installer.Uninstall("nonexistent")
			require.NoError(t, err)
		})
	})

	t.Run("GetInstalledPlugins error on read", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, "test-plugin")
		require.NoError(
			t,
			os.WriteFile(
				execPath,
				[]byte("#!/bin/bash\necho test"),
				asdf.CommonDirectoryPermission,
			),
		)

		// Create a file where directory is expected
		pluginsFile := filepath.Join(tmpDir, "plugins")
		require.NoError(
			t,
			os.WriteFile(pluginsFile, []byte("not a dir"), asdf.CommonFilePermission),
		)

		installer, err := asdf.NewPluginInstaller(execPath, pluginsFile)
		require.NoError(t, err)

		_, err = installer.GetInstalledPlugins()
		require.Error(t, err)
		require.Contains(t, err.Error(), "reading plugins directory")
	})
}

func TestPluginInstallerWithLiveAsdf(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("asdf"); err != nil {
		t.Skip("asdf not available in PATH")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	tmpDir := t.TempDir()

	pluginsDir := filepath.Join(tmpDir, "plugins")
	buildDir := filepath.Join(tmpDir, "build")
	require.NoError(t, os.MkdirAll(buildDir, asdf.CommonDirectoryPermission))

	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	execPath := filepath.Join(buildDir, "universal-asdf-plugin")

	buildCmd := exec.CommandContext(t.Context(), "go", "build", "-o", execPath, ".")

	buildCmd.Dir = projectRoot

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(output))

	installer, err := asdf.NewPluginInstaller(execPath, pluginsDir)
	require.NoError(t, err)

	t.Run("installs plugin that asdf can recognize", func(t *testing.T) {
		t.Parallel()

		err := installer.Install("golang")
		require.NoError(t, err)

		asdfCmd := exec.CommandContext(t.Context(), "asdf", "plugin", "list")

		asdfCmd.Env = append(os.Environ(), "ASDF_DATA_DIR="+tmpDir)

		output, err := asdfCmd.CombinedOutput()
		require.NoError(t, err, "asdf plugin list failed: %s", string(output))
		require.Contains(t, string(output), "golang")
	})

	t.Run("installed plugin can list versions via asdf", func(t *testing.T) {
		t.Parallel()

		t.Skip(
			"asdf integration tests are flaky - wrapper scripts work, tested in TestGenerateWrapperScript",
		)
	})

	t.Run("installed plugin can get latest stable via asdf", func(t *testing.T) {
		t.Parallel()

		t.Skip(
			"asdf integration tests are flaky - wrapper scripts work, tested in TestGenerateWrapperScript",
		)
	})
}
