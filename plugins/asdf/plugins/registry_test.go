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

package plugins

import (
	"os"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// TestRegistryPlugins runs standard plugin behavior tests for all registered plugins.
// This test suite traverses the plugin registry and validates that each plugin:
// - Can be instantiated via the registry
// - Implements the asdf.Plugin interface
// - Provides non-empty help information.
func TestRegistryPlugins(t *testing.T) {
	t.Parallel()

	registry := GetPluginRegistry()
	require.NotNil(t, registry, "expected plugin registry to be initialized")

	entries := registry.All()
	require.NotEmpty(t, entries, "expected registry to contain plugins")

	// Test each plugin entry
	for _, entry := range entries {
		// capture for parallel subtest
		for _, name := range entry.Names {
			// capture for parallel subtest
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				// Test that plugin can be retrieved by name
				plugin, err := GetPlugin(name)
				require.NoError(t, err, "expected plugin %s to be retrievable", name)
				require.NotNil(t, plugin, "expected plugin %s to be non-nil", name)

				// Test that plugin implements asdf.Plugin interface
				require.Implements(t, (*asdf.Plugin)(nil), plugin, "expected %s to implement asdf.Plugin", name)

				// Test that plugin provides help information (at minimum overview)
				help := plugin.Help()
				require.NotEmpty(t, help.Overview, "expected %s to provide help overview", name)

				// Test that plugin provides basic interface methods
				require.NotEmpty(t, plugin.Name(), "expected %s to provide name", name)
				require.NotEmpty(t, plugin.ListBinPaths(), "expected %s to provide bin paths", name)
				// ListLegacyFilenames and ExecEnv may return nil/empty for some plugins
			})
		}
	}
}

// TestRegistryPluginsOnline runs online integration tests for all registered plugins.
func TestRegistryPluginsOnline(t *testing.T) {
	t.Parallel()

	registry := GetPluginRegistry()
	require.NotNil(t, registry, "expected plugin registry to be initialized")

	entries := registry.All()
	require.NotEmpty(t, entries, "expected registry to contain plugins")

	// Test each plugin entry with online operations
	for _, entry := range entries {
		if len(entry.Names) == 0 {
			continue
		}

		name := entry.Names[0]
		t.Run(name+"_online", func(t *testing.T) {
			t.Parallel()

			plugin, err := GetPlugin(name)
			require.NoError(t, err, "expected plugin %s to be retrievable", name)
			require.NotNil(t, plugin, "expected plugin %s to be non-nil", name)

			ctx := t.Context()

			// Test ListAll with goldie snapshot
			t.Run("ListAll", func(t *testing.T) {
				versions, err := plugin.ListAll(ctx)
				if err != nil {
					t.Skipf("ListAll failed for %s: %v", name, err)
				}

				require.NotEmpty(t, versions, "expected %s to return versions", name)
			})

			// Test LatestStable with goldie snapshot
			t.Run("LatestStable", func(t *testing.T) {
				version, err := plugin.LatestStable(ctx, "")
				if err != nil {
					t.Skipf("LatestStable failed for %s: %v", name, err)
				}

				require.NotEmpty(t, version, "expected %s to return latest version", name)
			})
		})
	}
}

// TestRegistryUnknownPlugin verifies that unknown plugins return an error.
func TestRegistryUnknownPlugin(t *testing.T) {
	t.Parallel()

	plugin, err := GetPlugin("this-plugin-does-not-exist")
	require.Nil(t, plugin, "expected unknown plugin to return nil")
	require.Error(t, err, "expected unknown plugin to return error")
}

// TestRegistryGetPluginRegistry verifies that the registry can be retrieved.
func TestRegistryGetPluginRegistry(t *testing.T) {
	t.Parallel()

	registry := GetPluginRegistry()
	require.NotNil(t, registry, "expected registry to be non-nil")

	entries := registry.All()
	require.NotEmpty(t, entries, "expected registry to contain plugins")
}

// TestRegistryPluginAliases verifies that plugin aliases work correctly.
func TestRegistryPluginAliases(t *testing.T) {
	t.Parallel()

	// Test some known aliases
	aliases := map[string]string{
		"go":   "golang",
		"gh":   "github-cli",
		"node": "nodejs",
	}

	for alias, canonical := range aliases {
		t.Run(alias+"_alias", func(t *testing.T) {
			t.Parallel()

			pluginByAlias, err := GetPlugin(alias)
			require.NoError(t, err, "expected alias %s to be retrievable", alias)
			require.NotNil(t, pluginByAlias, "expected alias %s to return non-nil plugin", alias)

			pluginByCanonical, err := GetPlugin(canonical)
			require.NoError(t, err, "expected canonical %s to be retrievable", canonical)
			require.NotNil(t, pluginByCanonical, "expected canonical %s to return non-nil plugin", canonical)

			// Both should have the same name
			require.Equal(t, pluginByAlias.Name(), pluginByCanonical.Name(),
				"expected alias %s and canonical %s to have same name", alias, canonical)
		})
	}
}

// TestRegistryPluginsGoldie tests all plugins with goldie snapshots for ListAll and LatestStable.
// Run with -update to update snapshots: go test ./plugins/asdf/plugins -run TestRegistryPluginsGoldie -update
// Filter by plugin: PLUGIN=kubectl go test ./plugins/asdf/plugins -run TestRegistryPluginsGoldie.
func TestRegistryPluginsGoldie(t *testing.T) {
	t.Parallel()

	registry := GetPluginRegistry()
	require.NotNil(t, registry)

	entries := registry.All()
	require.NotEmpty(t, entries)

	pluginFilter := os.Getenv("PLUGIN")

	goldieTester := goldie.New(t, goldie.WithTestNameForDir(false))

	for _, entry := range entries {
		if len(entry.Names) == 0 {
			continue
		}

		name := entry.Names[0]

		if pluginFilter != "" && name != pluginFilter {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			plugin, err := GetPlugin(name)
			require.NoError(t, err)
			require.NotNil(t, plugin)

			ctx := t.Context()

			t.Run("list_all", func(t *testing.T) {
				t.Parallel()

				versions, err := plugin.ListAll(ctx)
				if err != nil {
					t.Skipf("ListAll failed for %s: %v", name, err)
				}

				require.NotEmpty(t, versions)

				versionData := strings.Join(versions, "\n")
				goldieTester.Assert(t, name+"_list_all", []byte(versionData))
			})

			t.Run("latest_stable", func(t *testing.T) {
				t.Parallel()

				version, err := plugin.LatestStable(ctx, "")
				if err != nil {
					t.Skipf("LatestStable failed for %s: %v", name, err)
				}

				require.NotEmpty(t, version)

				goldieTester.Assert(t, name+"_latest_stable", []byte(version))
			})
		})
	}
}

// TestRegistryPluginDownloadInstall tests a single plugin's download and install.
// Usage: PLUGIN=jq go test ./plugins/asdf/plugins -run TestRegistryPluginDownloadInstall.
func TestRegistryPluginDownloadInstall(t *testing.T) {
	t.Parallel()

	pluginName := os.Getenv("PLUGIN")
	if pluginName == "" {
		t.Skip("Set PLUGIN=<name> to test specific plugin download/install")
	}

	plugin, err := GetPlugin(pluginName)
	require.NoError(t, err)
	require.NotNil(t, plugin)

	ctx := t.Context()

	version, err := plugin.LatestStable(ctx, "")
	require.NoError(t, err, "Failed to get latest stable version")
	require.NotEmpty(t, version)

	t.Logf("Testing %s version %s", pluginName, version)

	downloadPath := t.TempDir()
	installPath := t.TempDir()

	t.Run("download", func(t *testing.T) {
		t.Parallel()

		err := plugin.Download(ctx, version, downloadPath)
		require.NoError(t, err, "Download should succeed")
	})

	t.Run("install", func(t *testing.T) {
		t.Parallel()

		err := plugin.Install(ctx, version, downloadPath, installPath)
		require.NoError(t, err, "Install should succeed")
	})

	t.Logf("Successfully downloaded and installed %s %s", pluginName, version)
}
