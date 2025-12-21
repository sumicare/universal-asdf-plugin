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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errSourceBuildNoVersionsFound is returned when no versions are discovered.
	errSourceBuildNoVersionsFound = errors.New("no versions found")
	// errSourceBuildNoVersionsMatching is returned when no versions match a LatestStable query.
	errSourceBuildNoVersionsMatching = errors.New("no versions matching query")
	// errSourceBuildArchiveMissing is returned when an expected source archive is missing.
	errSourceBuildArchiveMissing = errors.New("source archive missing")
	// errSourceBuildExtractedDirMissing is returned when the extracted source directory cannot be found.
	errSourceBuildExtractedDirMissing = errors.New("extracted source directory missing")
	// errSourceBuildArtifactMissing is returned when expected install artifacts cannot be found.
	errSourceBuildArtifactMissing = errors.New("install artifact missing")
	// errSourceBuildNoBuildStep is returned when no build step is configured.
	errSourceBuildNoBuildStep = errors.New("no build step configured")
	// errSourceBuildUnsupportedArchiveType is returned when the archive type is not supported.
	errSourceBuildUnsupportedArchiveType = errors.New("unsupported archive type")
)

type (
	// SourceBuildPlugin implements a generic asdf.Plugin for tools built from source archives.
	SourceBuildPlugin struct {
		Github *github.Client
		Config *SourceBuildPluginConfig
	}

	// SourceBuildPluginConfig configures the SourceBuildPlugin.
	SourceBuildPluginConfig struct {
		MinArchiveSize           *int64
		PostInstallVersion       func(ctx context.Context, version, installPath string) error
		BuildVersion             func(ctx context.Context, version, sourceDir, installPath string) error
		PreBuildVersion          func(ctx context.Context, version, sourceDir string) error
		SourceURLFunc            func(ctx context.Context, version string) (string, error)
		DownloadFile             func(ctx context.Context, url, destPath string) error
		CreateBinDir             *bool
		Help                     PluginHelp
		BinDir                   string
		VersionFilter            string
		ExtractedDirNameTemplate string
		RepoOwner                string
		RepoName                 string
		VersionPrefix            string
		ArchiveType              string
		Name                     string
		ArchiveNameTemplate      string
		SourceURLTemplate        string
		LegacyFilenames          []string
		ExpectedArtifacts        []string
		UseTags                  bool
		SkipExtract              bool
		SkipDownload             bool
		AutoDetectExtractedDir   bool
	}
)

// NewSourceBuildPlugin creates a new SourceBuildPlugin.
func NewSourceBuildPlugin(config *SourceBuildPluginConfig) *SourceBuildPlugin {
	if config.VersionPrefix == "" {
		config.VersionPrefix = "v"
	}

	if config.SourceURLTemplate == "" {
		config.SourceURLTemplate = "https://github.com/{{.RepoOwner}}/{{.RepoName}}/archive/refs/tags/{{.VersionPrefix}}{{.Version}}.tar.gz"
	}

	if config.ArchiveType == "" {
		config.ArchiveType = "tar.gz"
	}

	if config.ArchiveNameTemplate == "" {
		config.ArchiveNameTemplate = "{{.RepoName}}-{{.Version}}." + config.ArchiveType
	}

	if config.ExtractedDirNameTemplate == "" {
		config.ExtractedDirNameTemplate = "{{.RepoName}}-{{.Version}}"
	}

	if config.BinDir == "" {
		config.BinDir = "bin"
	}

	if config.CreateBinDir == nil {
		create := true

		config.CreateBinDir = &create
	}

	if config.MinArchiveSize == nil {
		minSize := int64(1024)

		config.MinArchiveSize = &minSize
	}

	return &SourceBuildPlugin{
		Config: config,
		Github: github.NewClient(),
	}
}

// WithGithubClient sets the GitHub client.
func (plugin *SourceBuildPlugin) WithGithubClient(client *github.Client) {
	plugin.Github = client
}

// Name returns the plugin name.
func (plugin *SourceBuildPlugin) Name() string {
	return plugin.Config.Name
}

// ListAll lists all available versions.
func (plugin *SourceBuildPlugin) ListAll(ctx context.Context) ([]string, error) {
	return ListGitHubVersions(ctx, plugin.Github, &ListGitHubVersionsConfig{
		RepoOwner:     plugin.Config.RepoOwner,
		RepoName:      plugin.Config.RepoName,
		VersionPrefix: plugin.Config.VersionPrefix,
		VersionFilter: plugin.Config.VersionFilter,
		UseTags:       plugin.Config.UseTags,
	})
}

// LatestStable returns the latest stable version matching the query.
func (plugin *SourceBuildPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errSourceBuildNoVersionsFound
	}

	latest := LatestVersion(versions, query)
	if latest == "" {
		return "", fmt.Errorf("%w: %s", errSourceBuildNoVersionsMatching, query)
	}

	return latest, nil
}

// Download downloads the specified version.
//
// By default, SourceBuildPlugin performs download during Install so that plugins
// can remain compatible with existing source-build implementations.
func (*SourceBuildPlugin) Download(_ context.Context, _, _ string) error {
	return nil
}

// Install downloads, extracts, and builds the specified version.
func (plugin *SourceBuildPlugin) Install(
	ctx context.Context,
	version, downloadPath, installPath string,
) error {
	if plugin.Config.BuildVersion == nil {
		return errSourceBuildNoBuildStep
	}

	err := EnsureDir(installPath)
	if err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	createBinDir := plugin.Config.CreateBinDir == nil || *plugin.Config.CreateBinDir
	if createBinDir && plugin.Config.BinDir != "" && plugin.Config.BinDir != "." {
		binDir := filepath.Join(installPath, plugin.Config.BinDir)

		err := EnsureDir(binDir)
		if err != nil {
			return fmt.Errorf("creating bin directory: %w", err)
		}
	}

	if len(plugin.Config.ExpectedArtifacts) > 0 {
		allPresent := true

		for _, rel := range plugin.Config.ExpectedArtifacts {
			if _, err := os.Stat(filepath.Join(installPath, rel)); err != nil {
				allPresent = false

				break
			}
		}

		if allPresent {
			for _, rel := range plugin.Config.ExpectedArtifacts {
				p := filepath.Join(installPath, rel)
				if strings.HasPrefix(
					filepath.Clean(rel),
					plugin.Config.BinDir+string(os.PathSeparator),
				) {
					_ = os.Chmod(p, CommonExecutablePermission)
				}
			}

			return nil
		}
	}

	workDir := downloadPath

	cleanup := func() {}

	if workDir == "" {
		tmp, err := os.MkdirTemp("", "asdf-src-*")
		if err != nil {
			return fmt.Errorf("creating temporary directory: %w", err)
		}

		workDir = tmp
		cleanup = func() { _ = os.RemoveAll(tmp) }
	}

	defer cleanup()

	err = EnsureDir(workDir)
	if err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	sourceDir := workDir

	archiveName := renderSourceBuildTemplate(
		plugin.Config.ArchiveNameTemplate,
		plugin.Config,
		version,
	)
	archivePath := filepath.Join(workDir, archiveName)

	minSize := int64(1024)
	if plugin.Config.MinArchiveSize != nil {
		minSize = *plugin.Config.MinArchiveSize
	}

	if !plugin.Config.SkipDownload {
		err := plugin.downloadSource(ctx, version, workDir, archivePath, minSize)
		if err != nil {
			return err
		}
	}

	if !plugin.Config.SkipExtract {
		var err error

		sourceDir, err = plugin.extractSource(version, workDir, archivePath)
		if err != nil {
			return err
		}
	}

	if plugin.Config.PreBuildVersion != nil {
		err := plugin.Config.PreBuildVersion(ctx, version, sourceDir)
		if err != nil {
			return err
		}
	}

	err = plugin.Config.BuildVersion(ctx, version, sourceDir, installPath)
	if err != nil {
		return err
	}

	if plugin.Config.PostInstallVersion != nil {
		err := plugin.Config.PostInstallVersion(ctx, version, installPath)
		if err != nil {
			return err
		}
	}

	for _, rel := range plugin.Config.ExpectedArtifacts {
		p := filepath.Join(installPath, rel)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("%w: %s", errSourceBuildArtifactMissing, rel)
		}

		if strings.HasPrefix(filepath.Clean(rel), plugin.Config.BinDir+string(os.PathSeparator)) {
			_ = os.Chmod(p, CommonExecutablePermission)
		}
	}

	return nil
}

// ListBinPaths returns the relative paths to directories containing binaries.
func (plugin *SourceBuildPlugin) ListBinPaths() string {
	return plugin.Config.BinDir
}

// ExecEnv returns environment variables to set when executing tool binaries.
func (*SourceBuildPlugin) ExecEnv(string) map[string]string {
	return make(map[string]string)
}

// Uninstall removes the specified version.
func (*SourceBuildPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// ListLegacyFilenames returns filenames to check for legacy version files.
func (plugin *SourceBuildPlugin) ListLegacyFilenames() []string {
	return plugin.Config.LegacyFilenames
}

// ParseLegacyFile parses a legacy version file and returns the version.
func (*SourceBuildPlugin) ParseLegacyFile(path string) (string, error) {
	return ReadLegacyVersionFile(path)
}

// Help returns help information for the plugin.
func (plugin *SourceBuildPlugin) Help() PluginHelp {
	return plugin.Config.Help
}

// downloadSource downloads the source archive if it doesn't exist or is too small.
func (plugin *SourceBuildPlugin) downloadSource(
	ctx context.Context,
	version, workDir, archivePath string,
	minSize int64,
) error {
	if info, err := os.Stat(archivePath); err == nil && info.Size() > minSize {
		return nil
	}

	var srcURL string

	if plugin.Config.SourceURLFunc != nil {
		resolved, err := plugin.Config.SourceURLFunc(ctx, version)
		if err != nil {
			return err
		}

		srcURL = resolved
	} else {
		srcURL = renderSourceBuildTemplate(plugin.Config.SourceURLTemplate, plugin.Config, version)
	}

	Msgf("Downloading %s %s source from %s", plugin.Config.Name, version, srcURL)

	downloadDest := archivePath

	if parsed, parseErr := url.Parse(srcURL); parseErr == nil {
		if base := filepath.Base(parsed.Path); base != "" && base != "." && base != "/" {
			downloadDest = filepath.Join(workDir, base)
		}
	}

	downloadFunc := plugin.Config.DownloadFile
	if downloadFunc == nil {
		downloadFunc = DownloadFile
	}

	err := downloadFunc(ctx, srcURL, downloadDest)
	if err != nil {
		return fmt.Errorf("downloading source: %w", err)
	}

	if downloadDest != archivePath {
		_ = os.RemoveAll(archivePath)

		err := os.Rename(downloadDest, archivePath)
		if err != nil {
			return fmt.Errorf("finalizing downloaded archive: %w", err)
		}
	}

	return nil
}

// extractSource extracts the source archive and returns the path to the extracted directory.
func (plugin *SourceBuildPlugin) extractSource(
	version, workDir, archivePath string,
) (string, error) {
	if _, err := os.Stat(archivePath); err != nil {
		return "", fmt.Errorf("%w: %s", errSourceBuildArchiveMissing, archivePath)
	}

	srcRoot := filepath.Join(workDir, "src")

	_ = os.RemoveAll(srcRoot)

	err := EnsureDir(srcRoot)
	if err != nil {
		return "", fmt.Errorf("creating source directory: %w", err)
	}

	switch plugin.Config.ArchiveType {
	case "tar.gz":
		err := ExtractTarGz(archivePath, srcRoot)
		if err != nil {
			return "", fmt.Errorf("extracting tar.gz: %w", err)
		}

	case "tar.xz":
		err := ExtractTarXz(archivePath, srcRoot)
		if err != nil {
			return "", fmt.Errorf("extracting tar.xz: %w", err)
		}

	case "zip":
		err := ExtractZip(archivePath, srcRoot)
		if err != nil {
			return "", fmt.Errorf("extracting zip: %w", err)
		}

	default:
		return "", fmt.Errorf(
			"%w: %s",
			errSourceBuildUnsupportedArchiveType,
			plugin.Config.ArchiveType,
		)
	}

	extractedDir := renderSourceBuildTemplate(
		plugin.Config.ExtractedDirNameTemplate,
		plugin.Config,
		version,
	)

	candidate := filepath.Join(srcRoot, extractedDir)
	if plugin.Config.AutoDetectExtractedDir {
		entries, err := osReadDir(srcRoot)
		if err != nil {
			return "", err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				candidate = filepath.Join(srcRoot, entry.Name())

				break
			}
		}
	}

	if _, err := os.Stat(candidate); err != nil {
		return "", fmt.Errorf("%w: %s", errSourceBuildExtractedDirMissing, candidate)
	}

	return candidate, nil
}

// renderSourceBuildTemplate substitutes template placeholders with config values.
func renderSourceBuildTemplate(
	template string,
	cfg *SourceBuildPluginConfig,
	version string,
) string {
	out := template

	out = strings.ReplaceAll(out, "{{.RepoOwner}}", cfg.RepoOwner)
	out = strings.ReplaceAll(out, "{{.RepoName}}", cfg.RepoName)
	out = strings.ReplaceAll(out, "{{.Name}}", cfg.Name)
	out = strings.ReplaceAll(out, "{{.Version}}", version)
	out = strings.ReplaceAll(out, "{{.VersionPrefix}}", cfg.VersionPrefix)

	return out
}
