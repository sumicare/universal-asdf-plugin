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
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

const (
	// zigIndexDownloadURL is the URL of the Zig download index.
	zigIndexDownloadURL = "https://ziglang.org/download/index.json"
)

type (
	// ZigRelease represents a single platform release.
	ZigRelease struct {
		Tarball string `json:"tarball"`
		Shasum  string `json:"shasum"`
		Size    string `json:"size"`
	}

	// ZigPlugin implements the asdf.Plugin interface for Zig.
	ZigPlugin struct {
		*asdf.SourceBuildPlugin

		ZigIndexURL string
	}
)

// NewZigPlugin creates a new Zig plugin instance.
func NewZigPlugin() asdf.Plugin {
	createBinDir := false
	cfg := &asdf.SourceBuildPluginConfig{
		Name:                   "zig",
		BinDir:                 ".",
		CreateBinDir:           &createBinDir,
		SkipDownload:           true,
		ArchiveType:            "tar.xz",
		ArchiveNameTemplate:    "zig.tar.xz",
		AutoDetectExtractedDir: true,
		ExpectedArtifacts:      []string{"zig"},
		BuildVersion: func(_ context.Context, _, sourceDir, installPath string) error {
			return asdf.CopyDir(sourceDir, installPath)
		},
	}

	return &ZigPlugin{
		SourceBuildPlugin: asdf.NewSourceBuildPlugin(cfg),
		ZigIndexURL:       zigIndexDownloadURL,
	}
}

// Name returns the plugin name.
func (*ZigPlugin) Name() string {
	return "zig"
}

// ListBinPaths returns the binary paths for Zig installations.
func (*ZigPlugin) ListBinPaths() string {
	return "."
}

// ExecEnv returns environment variables for Zig execution.
func (*ZigPlugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version filenames for Zig.
func (*ZigPlugin) ListLegacyFilenames() []string {
	return make([]string, 0)
}

// ParseLegacyFile parses a legacy version file.
func (*ZigPlugin) ParseLegacyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// Uninstall removes a Zig installation.
func (*ZigPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Zig plugin.
func (*ZigPlugin) Help() asdf.PluginHelp {
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

// ListAll lists all available Zig versions.
func (plugin *ZigPlugin) ListAll(ctx context.Context) ([]string, error) {
	// Fetch Zig download index
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, plugin.ZigIndexURL, http.NoBody)
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

	var rawIndex map[string]map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawIndex); err != nil {
		return nil, err
	}

	index := make(map[string]map[string]ZigRelease)
	for version, platforms := range rawIndex {
		index[version] = make(map[string]ZigRelease)
		for platform, data := range platforms {
			var release ZigRelease
			if err := json.Unmarshal(data, &release); err != nil {
				continue
			}

			if release.Tarball != "" {
				index[version][platform] = release
			}
		}
	}

	// Extract versions
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
func (plugin *ZigPlugin) LatestStable(ctx context.Context, query string) (string, error) {
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
func (plugin *ZigPlugin) Download(ctx context.Context, version, downloadPath string) error {
	destPath := filepath.Join(downloadPath, "zig.tar.xz")
	if info, err := os.Stat(destPath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for zig %s", version)
		return nil
	}

	// Fetch Zig download index
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, plugin.ZigIndexURL, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", errZigFetchIndexFailed, resp.StatusCode)
	}

	var rawIndex map[string]map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawIndex); err != nil {
		return err
	}

	index := make(map[string]map[string]ZigRelease)
	for ver, platforms := range rawIndex {
		index[ver] = make(map[string]ZigRelease)
		for platform, data := range platforms {
			var release ZigRelease
			if err := json.Unmarshal(data, &release); err != nil {
				continue
			}

			if release.Tarball != "" {
				index[ver][platform] = release
			}
		}
	}

	platforms, ok := index[version]
	if !ok {
		return fmt.Errorf("%w: %s", errZigVersionNotFound, version)
	}

	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}

	platformKey := fmt.Sprintf("%s-%s", arch, runtime.GOOS)

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
func (plugin *ZigPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	tarballPath := filepath.Join(downloadPath, "zig.tar.xz")
	if _, err := os.Stat(tarballPath); err != nil {
		return err
	}

	if err := plugin.SourceBuildPlugin.Install(ctx, version, downloadPath, installPath); err != nil {
		return err
	}

	if downloadPath != "" {
		_ = os.RemoveAll(filepath.Join(downloadPath, "src"))
	}

	fmt.Printf("Zig %s installed successfully\n", version)

	return nil
}
