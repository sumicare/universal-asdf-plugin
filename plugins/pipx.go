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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// errPipxDownloadFailed indicates a non-success HTTP response when downloading pipx.
var errPipxDownloadFailed = errors.New("download failed")

const (
	// pipxDownloadURL is the format string for constructing pipx release download URLs.
	pipxDownloadURL = "https://github.com/pypa/pipx/releases/download/%s/pipx.pyz"
)

// PipxPlugin implements the asdf.Plugin interface for pipx.
type PipxPlugin struct {
	*asdf.SourceBuildPlugin
}

// NewPipxPlugin creates a new pipx plugin instance.
func NewPipxPlugin() asdf.Plugin {
	plugin := &PipxPlugin{}

	plugin.SourceBuildPlugin = asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name:              "pipx",
		RepoOwner:         "pypa",
		RepoName:          "pipx",
		BinDir:            "bin",
		SkipDownload:      true,
		SkipExtract:       true,
		ExpectedArtifacts: []string{"bin/pipx"},
		BuildVersion: func(_ context.Context, version, downloadPath, installPath string) error {
			srcPath := filepath.Join(downloadPath, "pipx.pyz")

			binDir := filepath.Join(installPath, "bin")
			if err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission); err != nil {
				return fmt.Errorf("creating bin directory: %w", err)
			}

			dstPath := filepath.Join(binDir, "pipx.pyz")

			srcFile, err := os.Open(srcPath)
			if err != nil {
				return fmt.Errorf("opening source file: %w", err)
			}
			defer srcFile.Close()

			dstFile, err := os.Create(dstPath)
			if err != nil {
				return fmt.Errorf("creating destination file: %w", err)
			}
			defer dstFile.Close()

			if _, err := io.Copy(dstFile, srcFile); err != nil {
				return fmt.Errorf("copying file: %w", err)
			}

			wrapperPath := filepath.Join(binDir, "pipx")
			wrapperContent := fmt.Sprintf(`#!/bin/sh
exec python3 "%s" "$@"
`, dstPath)

			if err := os.WriteFile(wrapperPath, []byte(wrapperContent), asdf.CommonExecutablePermission); err != nil {
				return fmt.Errorf("creating wrapper script: %w", err)
			}

			asdf.Msgf("pipx %s installed successfully", version)

			return nil
		},
	})

	return plugin
}

// Name returns the plugin name.
func (*PipxPlugin) Name() string {
	return "pipx"
}

// Dependencies returns the list of plugins that must be installed before pipx.
func (*PipxPlugin) Dependencies() []string {
	return []string{"python"}
}

// ListBinPaths returns the binary paths for pipx installations.
func (*PipxPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for pipx execution.
func (*PipxPlugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for pipx.
func (*PipxPlugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a legacy pipx version file.
func (*PipxPlugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes a pipx installation.
func (*PipxPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the pipx plugin.
func (*PipxPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `pipx - Install and Run Python Applications in Isolated Environments.
This plugin downloads the pipx.pyz file from GitHub releases.`,
		Deps: `Requires Python 3.8+ to be installed and available in PATH.`,
		Config: `Environment variables:
  PIPX_HOME - Override pipx home directory
  PIPX_BIN_DIR - Override pipx bin directory`,
		Links: `Homepage: https://pipx.pypa.io/
Documentation: https://pipx.pypa.io/stable/
Source: https://github.com/pypa/pipx`,
	}
}

// ListAll lists all available pipx versions.
func (plugin *PipxPlugin) ListAll(ctx context.Context) ([]string, error) {
	return plugin.SourceBuildPlugin.ListAll(ctx)
}

// LatestStable returns the latest stable pipx version.
func (plugin *PipxPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	return plugin.SourceBuildPlugin.LatestStable(ctx, query)
}

// Download downloads the specified pipx version.
func (*PipxPlugin) Download(ctx context.Context, version, downloadPath string) error {
	pyzPath := filepath.Join(downloadPath, "pipx.pyz")
	if info, err := os.Stat(pyzPath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for pipx %s", version)

		return nil
	}

	url := fmt.Sprintf(pipxDownloadURL, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading pipx: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", errPipxDownloadFailed, resp.StatusCode)
	}

	outFile, err := os.Create(pyzPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// Install installs pipx from the downloaded .pyz file.
func (p *PipxPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	// Download if needed
	err := p.Download(ctx, version, downloadPath)
	if err != nil {
		return err
	}

	return p.SourceBuildPlugin.Install(ctx, version, downloadPath, installPath)
}
