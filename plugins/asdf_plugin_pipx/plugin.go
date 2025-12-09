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

package asdf_plugin_pipx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_python"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errPipxNoVersionsFound is returned when no pipx versions are discovered.
	errPipxNoVersionsFound = errors.New("no versions found")
	// errPipxNoVersionsMatching is returned when no versions match a LatestStable query.
	errPipxNoVersionsMatching = errors.New("no versions matching query")
	// errPipxDownloadFailed indicates a non-success HTTP response when downloading pipx.
	errPipxDownloadFailed = errors.New("download failed")

	// installPythonToolchain installs the Python toolchain into an asdf-style
	// installs tree under ASDF_DATA_DIR or ~/.asdf using the Python plugin
	// implementation. It is a variable so that tests can replace it with a
	// fast stub to avoid performing real Python installs.
	installPythonToolchain = asdf_plugin_python.InstallPythonToolchain //nolint:gochecknoglobals // configurable in tests

	// ensureToolchains ensures that the required toolchains have entries in a
	// .tool-versions file. It is a variable so that tests can replace it with a
	// stub to exercise error paths without invoking the real asdf tooling.
	ensureToolchains = asdf.EnsureToolchains //nolint:gochecknoglobals // configurable in tests

	// httpClient is the HTTP client used for downloading pipx. It is a variable
	// so that tests can replace it with a stub client.
	httpClient interface { //nolint:gochecknoglobals // configurable in tests
		Do(*http.Request) (*http.Response, error)
	} = http.DefaultClient

	// newRequestFn creates HTTP requests. It is a variable so that tests can
	// force request creation failures.
	newRequestFn = http.NewRequestWithContext //nolint:gochecknoglobals // configurable in tests

	// copyFileFn copies data from src to dst. It wraps io.Copy so tests can
	// force copy failures when exercising error paths.
	copyFileFn = io.Copy //nolint:gochecknoglobals // configurable in tests

	// writeFileFn writes data to a file path. It wraps os.WriteFile so tests can
	// force write failures when exercising error paths.
	writeFileFn = os.WriteFile //nolint:gochecknoglobals // configurable in tests
)

const (
	// pipxGitRepoURL points to the upstream pipx GitHub repository.
	pipxGitRepoURL = "https://github.com/pypa/pipx"
	// pipxDownloadURL is the format string for constructing pipx release download URLs.
	pipxDownloadURL = "https://github.com/pypa/pipx/releases/download/%s/pipx.pyz"
)

// Plugin implements the asdf.Plugin interface for pipx.
type Plugin struct {
	githubClient *github.Client
	downloadURL  string
}

// New creates a new pipx plugin instance.
func New() *Plugin {
	return &Plugin{
		githubClient: github.NewClient(),
		downloadURL:  pipxDownloadURL,
	}
}

// NewWithClient creates a new pipx plugin with custom client (for testing).
func NewWithClient(client *github.Client, downloadURL string) *Plugin {
	return &Plugin{
		githubClient: client,
		downloadURL:  downloadURL,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "pipx"
}

// ListBinPaths returns the binary paths for pipx installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for pipx execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for pipx.
func (*Plugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a legacy pipx version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes a pipx installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the pipx plugin.
func (*Plugin) Help() asdf.PluginHelp {
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
func (p *Plugin) ListAll(ctx context.Context) ([]string, error) {
	releases, err := p.githubClient.GetReleases(ctx, pipxGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	versionRegex := regexp.MustCompile(`^\d+\.\d+\.\d+`)

	versions := make([]string, 0, len(releases))
	for _, tag := range releases {
		version := strings.TrimPrefix(tag, "v")
		if versionRegex.MatchString(version) {
			versions = append(versions, version)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return asdf.CompareVersions(versions[i], versions[j]) < 0
	})

	return versions, nil
}

// LatestStable returns the latest stable pipx version.
func (p *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := p.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errPipxNoVersionsFound
	}

	if query != "" {
		var filtered []string
		for _, v := range versions {
			if strings.HasPrefix(v, query) {
				filtered = append(filtered, v)
			}
		}

		versions = filtered
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("%w: %s", errPipxNoVersionsMatching, query)
	}

	return versions[len(versions)-1], nil
}

// Download downloads the specified pipx version.
func (plugin *Plugin) Download(ctx context.Context, version, downloadPath string) error {
	pyzPath := filepath.Join(downloadPath, "pipx.pyz")
	if info, err := os.Stat(pyzPath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for pipx %s", version)
		return nil
	}

	url := fmt.Sprintf(plugin.downloadURL, version)

	req, err := newRequestFn(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := httpClient.Do(req)
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
func (*Plugin) Install(ctx context.Context, _, downloadPath, installPath string) error {
	if err := ensureToolchains(ctx, "python"); err != nil {
		return err
	}

	if err := installPythonToolchain(ctx); err != nil {
		return err
	}

	binDir := filepath.Join(installPath, "bin")
	if err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	srcPath := filepath.Join(downloadPath, "pipx.pyz")
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

	if _, err := copyFileFn(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	wrapperPath := filepath.Join(binDir, "pipx")
	wrapperContent := fmt.Sprintf(`#!/bin/sh
exec python3 "%s" "$@"
`, dstPath)

	if err := writeFileFn(wrapperPath, []byte(wrapperContent), asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating wrapper script: %w", err)
	}

	return nil
}
