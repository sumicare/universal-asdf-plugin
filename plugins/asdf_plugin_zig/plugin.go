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

package asdf_plugin_zig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// errZigFetchIndexFailed indicates a non-success HTTP response when fetching the Zig index.
	errZigFetchIndexFailed = errors.New("failed to fetch zig index")
	// errZigNoVersionsFound is returned when no Zig versions are discovered.
	errZigNoVersionsFound = errors.New("no versions found")
	// errZigNoVersionsMatching is returned when no versions match a LatestStable query.
	errZigNoVersionsMatching = errors.New("no versions matching query")
	// errZigVersionNotFound is returned when the requested Zig version does not exist in the index.
	errZigVersionNotFound = errors.New("version not found")
	// errZigNoReleaseForPlatform is returned when no release exists for the current platform.
	errZigNoReleaseForPlatform = errors.New("no release for platform")
	// errZigNoDirInArchive is returned when no top-level directory is found after extraction.
	errZigNoDirInArchive = errors.New("no directory found in archive")

	// zigIndexDownloadURL is the URL used by fetchIndex to retrieve the Zig index.
	// It is defined as a variable so tests can override it with a mock server.
	zigIndexDownloadURL = zigIndexURL //nolint:gochecknoglobals // we're mutating this for testing, not thread-safe but meh
)

const (
	// zigIndexURL is the URL of the Zig download index.
	zigIndexURL = "https://ziglang.org/download/index.json"
)

type (
	// ZigRelease represents a single platform release.
	ZigRelease struct {
		Tarball string `json:"tarball"`
		Shasum  string `json:"shasum"`
		Size    string `json:"size"`
	}

	// Plugin implements the asdf.Plugin interface for Zig.
	Plugin struct{}
)

// New creates a new Zig plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "zig"
}

// ListBinPaths returns the binary paths for Zig installations.
func (*Plugin) ListBinPaths() string {
	return "."
}

// ExecEnv returns environment variables for Zig execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version filenames for Zig.
func (*Plugin) ListLegacyFilenames() []string {
	return make([]string, 0)
}

// ParseLegacyFile parses a legacy version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// Uninstall removes a Zig installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Zig plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Zig - A general-purpose programming language and toolchain for maintaining
robust, optimal, and reusable software.`,
		Deps:   `No additional dependencies required.`,
		Config: `No additional configuration required.`,
		Links: `Homepage: https://ziglang.org/
Documentation: https://ziglang.org/documentation/
Source: https://codeberg.org/ziglang/zig`,
	}
}

// getPlatformKey returns the platform key for the current platform.
func (*Plugin) getPlatformKey() string {
	return platformKeyFor(runtime.GOOS, runtime.GOARCH)
}

// platformKeyFor returns the platform key for a given OS/arch pair.
// It is a small helper to make getPlatformKey easier to test.
func platformKeyFor(goos, goarch string) string {
	arch := goarch
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}

	return fmt.Sprintf("%s-%s", arch, goos)
}

// fetchIndex fetches the Zig download index.
func (*Plugin) fetchIndex(ctx context.Context) (map[string]map[string]ZigRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zigIndexDownloadURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errZigFetchIndexFailed, resp.StatusCode)
	}

	var index map[string]map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]ZigRelease)
	for version, platforms := range index {
		result[version] = make(map[string]ZigRelease)
		for platform, data := range platforms {
			var release ZigRelease
			if err := json.Unmarshal(data, &release); err != nil {
				continue
			}

			if release.Tarball != "" {
				result[version][platform] = release
			}
		}
	}

	return result, nil
}

// ListAll lists all available Zig versions.
func (plugin *Plugin) ListAll(ctx context.Context) ([]string, error) {
	index, err := plugin.fetchIndex(ctx)
	if err != nil {
		return nil, err
	}

	var versions []string
	for version := range index {
		if version != "master" {
			versions = append(versions, version)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return asdf.CompareVersions(versions[i], versions[j]) < 0
	})

	return versions, nil
}

// LatestStable returns the latest stable Zig version.
func (plugin *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errZigNoVersionsFound
	}

	if query == "" {
		return versions[len(versions)-1], nil
	}

	// Filter by query prefix
	var filtered []string
	for _, v := range versions {
		if len(v) >= len(query) && v[:len(query)] == query {
			filtered = append(filtered, v)
		}
	}

	if len(filtered) == 0 {
		return "", fmt.Errorf("%w: %s", errZigNoVersionsMatching, query)
	}

	return filtered[len(filtered)-1], nil
}

// Download downloads the Zig tarball.
func (plugin *Plugin) Download(ctx context.Context, version, downloadPath string) error {
	destPath := filepath.Join(downloadPath, "zig.tar.xz")
	if info, err := os.Stat(destPath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for zig %s", version)
		return nil
	}

	index, err := plugin.fetchIndex(ctx)
	if err != nil {
		return err
	}

	platforms, ok := index[version]
	if !ok {
		return fmt.Errorf("%w: %s", errZigVersionNotFound, version)
	}

	platformKey := plugin.getPlatformKey()

	release, ok := platforms[platformKey]
	if !ok {
		return fmt.Errorf("%w: %s", errZigNoReleaseForPlatform, platformKey)
	}

	fmt.Printf("Downloading Zig %s from %s\n", version, release.Tarball)

	if err := asdf.DownloadFile(ctx, release.Tarball, destPath); err != nil {
		return fmt.Errorf("downloading zig: %w", err)
	}

	return nil
}

// Install installs Zig from the downloaded tarball.
func (*Plugin) Install(_ context.Context, version, downloadPath, installPath string) error {
	tarballPath := filepath.Join(downloadPath, "zig.tar.xz")

	tempDir, err := os.MkdirTemp("", "zig-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := asdf.ExtractTarXz(tarballPath, tempDir); err != nil {
		return fmt.Errorf("extracting zig: %w", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	var extractedDir string
	for _, entry := range entries {
		if entry.IsDir() {
			extractedDir = filepath.Join(tempDir, entry.Name())
			break
		}
	}

	if extractedDir == "" {
		return errZigNoDirInArchive
	}

	if err := os.MkdirAll(installPath, asdf.CommonDirectoryPermission); err != nil {
		return err
	}

	if err := copyDir(extractedDir, installPath); err != nil {
		return fmt.Errorf("copying files: %w", err)
	}

	fmt.Printf("Zig %s installed successfully\n", version)

	return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}

			return os.Symlink(link, dstPath)
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file.
func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	return err
}
