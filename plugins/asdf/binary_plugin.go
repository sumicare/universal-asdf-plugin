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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errUnsupportedPlatform is returned when the current OS is not supported.
	errUnsupportedPlatform = errors.New("unsupported platform")
	// errUnsupportedArchitecture is returned when the current CPU architecture is not supported.
	errUnsupportedArchitecture = errors.New("unsupported architecture")
	// errNoBinaryFound is returned when no candidate binary is found in the download directory.
	errNoBinaryFound = errors.New("no binary found")
	// errBinaryNotFoundInArchive is returned when the expected binary is missing from an extracted archive.
	errBinaryNotFoundInArchive = errors.New("binary not found in archive")
)

type (
	// BinaryPlugin implements a generic asdf.Plugin for GitHub release binaries.
	BinaryPlugin struct {
		Github *github.Client
		Config *BinaryPluginConfig
	}

	// BinaryPluginConfig configures the BinaryPlugin.
	BinaryPluginConfig struct {
		OsMap               map[string]string
		UseReleases         *bool
		ArchMap             map[string]string
		VersionPrefix       string
		FileNameTemplate    string
		DownloadURLTemplate string
		Name                string
		BinaryName          string
		RepoName            string
		HelpLink            string
		HelpDescription     string
		ArchiveType         string
		VersionFilter       string
		RepoOwner           string
	}
)

// NewBinaryPlugin creates a new GenericPlugin.
func NewBinaryPlugin(config *BinaryPluginConfig) *BinaryPlugin {
	if config.VersionPrefix == "" {
		config.VersionPrefix = "v"
	}

	if config.FileNameTemplate == "" {
		config.FileNameTemplate = "{{.BinaryName}}-{{.Platform}}-{{.Arch}}"
	}

	if config.DownloadURLTemplate == "" {
		config.DownloadURLTemplate = "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/v{{.Version}}/{{.FileName}}"
	}

	if config.OsMap == nil {
		config.OsMap = map[string]string{
			"darwin": "darwin",
			"linux":  "linux",
		}
	}

	if config.ArchMap == nil {
		config.ArchMap = map[string]string{
			"amd64": "amd64",
			"arm64": "arm64",
		}
	}

	return &BinaryPlugin{
		Config: config,
		Github: github.NewClient(),
	}
}

// WithGithubClient sets the GitHub client.
func (plugin *BinaryPlugin) WithGithubClient(client *github.Client) *BinaryPlugin {
	plugin.Github = client
	return plugin
}

// Name returns the plugin name.
func (plugin *BinaryPlugin) Name() string {
	return plugin.Config.Name
}

// ListAll lists all available versions.
func (plugin *BinaryPlugin) ListAll(ctx context.Context) ([]string, error) {
	repoURL := fmt.Sprintf("https://github.com/%s/%s", plugin.Config.RepoOwner, plugin.Config.RepoName)

	useReleases := plugin.Config.UseReleases == nil || *plugin.Config.UseReleases

	var (
		tags []string
		err  error
	)

	if useReleases {
		tags, err = plugin.Github.GetReleases(ctx, repoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}
	} else {
		tags, err = plugin.Github.GetTags(ctx, repoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list tags: %w", err)
		}
	}

	// Compile version filter regex if provided
	var versionFilter *regexp.Regexp
	if plugin.Config.VersionFilter != "" {
		versionFilter, err = regexp.Compile(plugin.Config.VersionFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid version filter regex: %w", err)
		}
	}

	effectivePrefix := plugin.Config.VersionPrefix

	versions := make([]string, 0, len(tags))
	for _, tag := range tags {
		if effectivePrefix != "" {
			if !useReleases && !strings.HasPrefix(tag, effectivePrefix) {
				continue
			}

			tag = strings.TrimPrefix(tag, effectivePrefix)
		}

		if tag == "" {
			continue
		}

		if versionFilter != nil && !versionFilter.MatchString(tag) {
			continue
		}

		versions = append(versions, tag)
	}

	return versions, nil
}

// Download downloads the specified version.
func (plugin *BinaryPlugin) Download(ctx context.Context, version, downloadPath string) error {
	platform, err := GetPlatform()
	if err != nil {
		return err
	}

	arch, err := GetArch()
	if err != nil {
		return err
	}

	mappedPlatform, ok := plugin.Config.OsMap[platform]
	if !ok {
		return fmt.Errorf("%w: %s", errUnsupportedPlatform, platform)
	}

	mappedArch, ok := plugin.Config.ArchMap[arch]
	if !ok {
		return fmt.Errorf("%w: %s", errUnsupportedArchitecture, arch)
	}

	fileName := plugin.Config.FileNameTemplate

	fileName = strings.ReplaceAll(fileName, "{{.Version}}", version)
	fileName = strings.ReplaceAll(fileName, "{{.Platform}}", mappedPlatform)
	fileName = strings.ReplaceAll(fileName, "{{.Arch}}", mappedArch)
	fileName = strings.ReplaceAll(fileName, "{{.BinaryName}}", plugin.Config.BinaryName)

	url := plugin.Config.DownloadURLTemplate

	url = strings.ReplaceAll(url, "{{.RepoOwner}}", plugin.Config.RepoOwner)
	url = strings.ReplaceAll(url, "{{.RepoName}}", plugin.Config.RepoName)
	url = strings.ReplaceAll(url, "{{.Version}}", version)
	url = strings.ReplaceAll(url, "{{.FileName}}", fileName)

	url = strings.ReplaceAll(url, "{{.Platform}}", mappedPlatform)
	url = strings.ReplaceAll(url, "{{.Arch}}", mappedArch)
	url = strings.ReplaceAll(url, "{{.BinaryName}}", plugin.Config.BinaryName)

	binaryPath := filepath.Join(downloadPath, fileName)

	if info, err := os.Stat(binaryPath); err == nil && info.Size() > 1024 {
		Msgf("Using cached download for %s %s", plugin.Config.Name, version)
		return nil
	}

	Msgf("Downloading %s %s from %s", plugin.Config.Name, version, url)

	if err := DownloadFile(ctx, url, binaryPath); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	if err := os.Chmod(binaryPath, CommonExecutablePermission); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

// Install installs the downloaded version.
func (plugin *BinaryPlugin) Install(_ context.Context, version, downloadPath, installPath string) error {
	entries, err := os.ReadDir(downloadPath)
	if err != nil {
		return err
	}

	var binaryName string
	for _, entry := range entries {
		if !entry.IsDir() {
			binaryName = entry.Name()
			break
		}
	}

	if binaryName == "" {
		return fmt.Errorf("%w in %s", errNoBinaryFound, downloadPath)
	}

	binaryPath := filepath.Join(downloadPath, binaryName)

	Msgf("Installing %s %s to %s", plugin.Config.Name, version, installPath)

	binDir := filepath.Join(installPath, "bin")
	if err := EnsureDir(binDir); err != nil {
		return err
	}

	destPath := filepath.Join(binDir, plugin.Config.BinaryName)

	switch plugin.Config.ArchiveType {
	case "gz":
		if err := ExtractGz(binaryPath, destPath); err != nil {
			return fmt.Errorf("failed to extract gz: %w", err)
		}

	case "tar.gz":
		if err := extractAndCopyBinary(binaryPath, destPath, plugin.Config.BinaryName, ExtractTarGz); err != nil {
			return err
		}

	case "tar.xz":
		if err := extractAndCopyBinary(binaryPath, destPath, plugin.Config.BinaryName, ExtractTarXz); err != nil {
			return err
		}

	case "zip":
		if err := extractAndCopyBinary(binaryPath, destPath, plugin.Config.BinaryName, ExtractZip); err != nil {
			return err
		}

	default:
		if err := copyFile(binaryPath, destPath); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}
	}

	if err := os.Chmod(destPath, CommonExecutablePermission); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	Msgf("%s %s installed successfully", plugin.Config.Name, version)

	return nil
}

// extractAndCopyBinary extracts an archive to a temp directory, finds the binary by name, and copies it to destPath.
func extractAndCopyBinary(archivePath, destPath, binaryName string, extractFn func(string, string) error) error {
	tempDir, err := os.MkdirTemp("", "asdf-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := extractFn(archivePath, tempDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	foundPath := ""

	if err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.Name() == binaryName {
			foundPath = path
			return filepath.SkipAll
		}

		return nil
	}); err != nil {
		return err
	}

	if foundPath == "" {
		return fmt.Errorf("%w: %s", errBinaryNotFoundInArchive, binaryName)
	}

	if err := copyFile(foundPath, destPath); err != nil {
		return fmt.Errorf("failed to copy binary from archive: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, CommonExecutablePermission)
}

// Uninstall removes the specified version.
func (*BinaryPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// ListBinPaths returns the list of binary paths.
func (*BinaryPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for execution.
func (*BinaryPlugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version file names.
func (*BinaryPlugin) ListLegacyFilenames() []string {
	return make([]string, 0)
}

// ParseLegacyFile parses a legacy version file.
func (*BinaryPlugin) ParseLegacyFile(path string) (string, error) {
	return ReadLegacyVersionFile(path)
}

// LatestStable returns the latest stable version.
func (plugin *BinaryPlugin) LatestStable(ctx context.Context, pattern string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	return LatestVersion(versions, pattern), nil
}

// Help returns help information for the plugin.
func (plugin *BinaryPlugin) Help() PluginHelp {
	return PluginHelp{
		Overview: fmt.Sprintf("%s - %s", plugin.Config.Name, plugin.Config.HelpDescription),
		Deps:     "No additional dependencies required",
		Config:   "No additional configuration required",
		Links: fmt.Sprintf(`Documentation: %s
GitHub: https://github.com/%s/%s`, plugin.Config.HelpLink, plugin.Config.RepoOwner, plugin.Config.RepoName),
	}
}
