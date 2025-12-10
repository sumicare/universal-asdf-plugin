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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

func TestBinaryPluginBasicMethods(t *testing.T) {
	t.Parallel()

	config := asdf.BinaryPluginConfig{
		Name:            "test-tool",
		RepoOwner:       "owner",
		RepoName:        "repo",
		BinaryName:      "test-tool",
		HelpDescription: "Test Description",
		HelpLink:        "http://example.com",
	}
	plugin := asdf.NewBinaryPlugin(&config)

	require.Equal(t, "test-tool", plugin.Name())
	require.Equal(t, "bin", plugin.ListBinPaths())
	require.Empty(t, plugin.ExecEnv("/some/path"))
	require.Empty(t, plugin.ListLegacyFilenames())

	help := plugin.Help()
	require.Contains(t, help.Overview, "test-tool - Test Description")
	require.Contains(t, help.Links, "http://example.com")
}

func TestBinaryPluginInstall(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	downloadPath := filepath.Join(tempDir, "download")
	installPath := filepath.Join(tempDir, "install")

	require.NoError(t, os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission))

	err := os.WriteFile(filepath.Join(downloadPath, "some-binary"), []byte("content"), asdf.CommonDirectoryPermission)
	require.NoError(t, err)

	config := asdf.BinaryPluginConfig{
		Name:       "test-tool",
		RepoOwner:  "owner",
		RepoName:   "repo",
		BinaryName: "test-tool",
	}
	plugin := asdf.NewBinaryPlugin(&config)

	err = plugin.Install(t.Context(), "1.0.0", downloadPath, installPath)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(installPath, "bin", "test-tool"))
	require.NoError(t, err)
}

func TestBinaryPluginUninstall(t *testing.T) {
	t.Parallel()

	config := asdf.BinaryPluginConfig{
		Name:       "test-tool",
		RepoOwner:  "owner",
		RepoName:   "repo",
		BinaryName: "test-tool",
	}
	plugin := asdf.NewBinaryPlugin(&config)

	tempDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "file"), []byte("content"), asdf.CommonFilePermission)
	require.NoError(t, err)

	err = plugin.Uninstall(t.Context(), tempDir)
	require.NoError(t, err)

	_, err = os.Stat(tempDir)
	require.True(t, os.IsNotExist(err))
}
