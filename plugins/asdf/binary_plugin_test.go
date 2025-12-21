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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// createTestTarGz creates a tar.gz archive containing a single file.
func createTestTarGz(t *testing.T, archivePath, fileName, content string) {
	t.Helper()

	file, err := os.Create(archivePath)
	require.NoError(t, err)

	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	header := &tar.Header{
		Name: fileName,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	require.NoError(t, tarWriter.WriteHeader(header))

	_, err = tarWriter.Write([]byte(content))
	require.NoError(t, err)
}

// createTestZip creates a zip archive containing a single file.
func createTestZip(t *testing.T, archivePath, fileName, content string) {
	t.Helper()

	file, err := os.Create(archivePath)
	require.NoError(t, err)

	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	writer, err := zipWriter.Create(fileName)
	require.NoError(t, err)

	_, err = writer.Write([]byte(content))
	require.NoError(t, err)
}

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

	err := os.WriteFile(
		filepath.Join(downloadPath, "some-binary"),
		[]byte("content"),
		asdf.CommonDirectoryPermission,
	)
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
	err := os.WriteFile(
		filepath.Join(tempDir, "file"),
		[]byte("content"),
		asdf.CommonFilePermission,
	)
	require.NoError(t, err)

	err = plugin.Uninstall(t.Context(), tempDir)
	require.NoError(t, err)

	_, err = os.Stat(tempDir)
	require.True(t, os.IsNotExist(err))
}

func TestBinaryPluginWithGithubClient(t *testing.T) {
	t.Parallel()

	config := asdf.BinaryPluginConfig{
		Name:       "test-tool",
		RepoOwner:  "owner",
		RepoName:   "repo",
		BinaryName: "test-tool",
	}
	plugin := asdf.NewBinaryPlugin(&config)

	// WithGithubClient should return the same plugin for chaining
	result := plugin.WithGithubClient(nil)
	require.Same(t, plugin, result)
}

func TestBinaryPluginParseLegacyFile(t *testing.T) {
	t.Parallel()

	config := asdf.BinaryPluginConfig{
		Name:       "test-tool",
		RepoOwner:  "owner",
		RepoName:   "repo",
		BinaryName: "test-tool",
	}
	plugin := asdf.NewBinaryPlugin(&config)

	t.Run("parses valid legacy file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".test-version")
		err := os.WriteFile(legacyFile, []byte("1.2.3\n"), asdf.CommonFilePermission)
		require.NoError(t, err)

		version, err := plugin.ParseLegacyFile(legacyFile)
		require.NoError(t, err)
		require.Equal(t, "1.2.3", version)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		t.Parallel()

		_, err := plugin.ParseLegacyFile("/nonexistent/path")
		require.Error(t, err)
	})
}

func runBinaryPluginInstallWithArchive(
	t *testing.T,
	archiveFilename string,
	createArchive func(t *testing.T, archivePath string),
) {
	t.Helper()

	tempDir := t.TempDir()
	downloadPath := filepath.Join(tempDir, "download")
	installPath := filepath.Join(tempDir, "install")

	require.NoError(t, os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission))

	archivePath := filepath.Join(downloadPath, archiveFilename)
	createArchive(t, archivePath)

	config := asdf.BinaryPluginConfig{
		Name:       "test-tool",
		RepoOwner:  "owner",
		RepoName:   "repo",
		BinaryName: "test-tool",
	}
	plugin := asdf.NewBinaryPlugin(&config)

	err := plugin.Install(t.Context(), "1.0.0", downloadPath, installPath)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(installPath, "bin", "test-tool"))
	require.NoError(t, err)
}

func TestBinaryPluginInstallWithArchive(t *testing.T) {
	t.Parallel()

	runBinaryPluginInstallWithArchive(
		t,
		"test-tool.tar.gz",
		func(t *testing.T, archivePath string) {
			t.Helper()

			createTestTarGz(t, archivePath, "test-tool", "binary content")
		},
	)
}

func TestBinaryPluginInstallWithZipArchive(t *testing.T) {
	t.Parallel()

	runBinaryPluginInstallWithArchive(t, "test-tool.zip", func(t *testing.T, archivePath string) {
		t.Helper()

		createTestZip(t, archivePath, "test-tool", "binary content")
	})
}

func TestBinaryPluginInstallWithGzFile(t *testing.T) {
	t.Parallel()

	runBinaryPluginInstallWithArchive(t, "test-tool.gz", func(t *testing.T, archivePath string) {
		t.Helper()

		createTestGz(t, archivePath, "binary content")
	})
}

// createTestGz creates a gzip file containing content.
func createTestGz(t *testing.T, archivePath, content string) {
	t.Helper()

	file, err := os.Create(archivePath)
	require.NoError(t, err)

	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	_, err = gzWriter.Write([]byte(content))
	require.NoError(t, err)
}
