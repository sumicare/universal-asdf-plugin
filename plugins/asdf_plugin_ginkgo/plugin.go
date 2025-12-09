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

package asdf_plugin_ginkgo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_go"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errGinkgoNoVersionsFound is returned when no Ginkgo versions are discovered.
	errGinkgoNoVersionsFound = errors.New("no versions found")
	// errGinkgoNoVersionsMatching is returned when no versions match a LatestStable query.
	errGinkgoNoVersionsMatching = errors.New("no versions matching query")
	// errGinkgoBinaryNotFound is returned when the installed ginkgo binary cannot be located.
	errGinkgoBinaryNotFound = errors.New("ginkgo binary not found after installation")

	// mkdirAllFn is a test seam for creating directories.
	// It is used to create directories during the installation process.
	mkdirAllFn = os.MkdirAll //nolint:gochecknoglobals // test seams configurable in tests
	// downloadFileFn is a test seam for downloading source archives.
	// It is used to download the Ginkgo source archive during the installation process.
	downloadFileFn = asdf.DownloadFile //nolint:gochecknoglobals // test seams configurable in tests
	// execCommandContextFn is a test seam for running external commands.
	// It is used to run the Go build command during the installation process.
	execCommandContextFn = exec.CommandContext //nolint:gochecknoglobals // test seams configurable in tests
	// statFn is a test seam for querying file metadata.
	// It is used to check if the installed ginkgo binary exists.
	statFn = os.Stat //nolint:gochecknoglobals // test seams configurable in tests
)

// Plugin implements the asdf.Plugin interface for Ginkgo.
type Plugin struct {
	githubClient *github.Client
}

// New creates a new Ginkgo plugin instance.
func New() *Plugin {
	return &Plugin{
		githubClient: github.NewClient(),
	}
}

// NewWithClient creates a new Ginkgo plugin with a custom GitHub client.
func NewWithClient(client *github.Client) *Plugin {
	return &Plugin{
		githubClient: client,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "ginkgo"
}

// ListBinPaths returns the binary paths for Ginkgo installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Ginkgo execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version filenames for Ginkgo.
func (*Plugin) ListLegacyFilenames() []string {
	return make([]string, 0)
}

// ParseLegacyFile parses a legacy version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// Uninstall removes a Ginkgo installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Ginkgo plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Ginkgo - A BDD-style Go testing framework.
Ginkgo is built from the official source archive using Go, which requires Go to be installed.`,
		Deps:   `Requires Go to be installed and available in PATH.`,
		Config: `No additional configuration required.`,
		Links: `Homepage: https://onsi.github.io/ginkgo/
Source: https://github.com/onsi/ginkgo`,
	}
}

// ListAll lists all available Ginkgo versions from GitHub tags.
func (p *Plugin) ListAll(ctx context.Context) ([]string, error) {
	tags, err := p.githubClient.GetTags(ctx, "https://github.com/onsi/ginkgo")
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "v2.") {
			versions = append(versions, strings.TrimPrefix(tag, "v"))
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return asdf.CompareVersions(versions[i], versions[j]) < 0
	})

	return versions, nil
}

// LatestStable returns the latest stable Ginkgo version.
func (p *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := p.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errGinkgoNoVersionsFound
	}

	if query == "" {
		return versions[len(versions)-1], nil
	}

	// Filter by query prefix
	var filtered []string
	for _, v := range versions {
		if strings.HasPrefix(v, query) {
			filtered = append(filtered, v)
		}
	}

	if len(filtered) == 0 {
		return "", fmt.Errorf("%w: %s", errGinkgoNoVersionsMatching, query)
	}

	return filtered[len(filtered)-1], nil
}

// Download is a no-op for Ginkgo since installation downloads the source archive directly.
func (*Plugin) Download(_ context.Context, _, _ string) error {
	return nil
}

// Install downloads the Ginkgo source archive and builds the ginkgo CLI using go build.
func (*Plugin) Install(ctx context.Context, version, _, installPath string) error {
	if err := asdf_plugin_go.EnsureGoToolchainEntries(ctx); err != nil {
		return err
	}

	if err := asdf_plugin_go.InstallGoToolchain(ctx); err != nil {
		return err
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go is required to install ginkgo but was not found in PATH: %w", err)
	}

	binDir := filepath.Join(installPath, "bin")
	if err := mkdirAllFn(binDir, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "ginkgo-src-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	archiveName := fmt.Sprintf("ginkgo-%s.tar.gz", version)
	archivePath := filepath.Join(tempDir, archiveName)
	sourceDir := filepath.Join(tempDir, "ginkgo-"+version)
	url := fmt.Sprintf("https://github.com/onsi/ginkgo/archive/refs/tags/v%s.tar.gz", version)

	asdf.Msgf("Downloading ginkgo %s source from %s", version, url)

	if err := downloadFileFn(ctx, url, archivePath); err != nil {
		return fmt.Errorf("downloading ginkgo source: %w", err)
	}

	tarCmd := execCommandContextFn(ctx, "tar", "-xzf", archivePath, "-C", tempDir)

	tarCmd.Stdout = os.Stderr

	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("extracting ginkgo source: %w", err)
	}

	buildCmd := execCommandContextFn(ctx, goPath, "build", "-o", filepath.Join(binDir, "ginkgo"), "./ginkgo")

	buildCmd.Dir = sourceDir
	buildCmd.Stdout = os.Stderr
	buildCmd.Stderr = os.Stderr
	buildCmd.Env = os.Environ()

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("building ginkgo: %w", err)
	}

	binaryPath := filepath.Join(binDir, "ginkgo")
	if _, err := statFn(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", errGinkgoBinaryNotFound, version)
	}

	fmt.Printf("Ginkgo %s installed successfully\n", version)

	return nil
}
