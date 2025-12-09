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

package asdf_plugin_argo

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
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_nodejs"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errArgoNoVersionsFound is returned when no Argo versions are discovered.
	errArgoNoVersionsFound = errors.New("no versions found")
	// errArgoNoVersionsMatching is returned when no versions match a LatestStable query.
	errArgoNoVersionsMatching = errors.New("no versions matching query")
	// errArgoBinaryNotFound is returned when the installed argo binary cannot be located.
	errArgoBinaryNotFound = errors.New("argo binary not found after installation")

	// installBuildToolchainsFunc is a testable hook for the build toolchains installation.
	installBuildToolchainsFunc = installBuildToolchains //nolint:gochecknoglobals // test hook overridden in tests

	// downloadFileFn is a test seam for the file download helper used during installation.
	downloadFileFn = asdf.DownloadFile //nolint:gochecknoglobals // test seam configurable in tests
	// execCommandContextFn is a test seam for constructing external commands during installation.
	execCommandContextFn = exec.CommandContext //nolint:gochecknoglobals // test seam configurable in tests
	// statFn is a test seam for filesystem stat checks used to verify the installed binary.
	statFn = os.Stat //nolint:gochecknoglobals // test seam configurable in tests
	// ensureToolchainsFn is a test seam for ensuring required toolchain entries exist.
	ensureToolchainsFn = asdf.EnsureToolchains //nolint:gochecknoglobals // test seam configurable in tests
	// ensureToolVersionsFileFn is a test seam for creating or updating .tool-versions files.
	ensureToolVersionsFileFn = asdf.EnsureToolVersionsFile //nolint:gochecknoglobals // test seam configurable in tests
)

// Plugin implements the asdf.Plugin interface for Argo Workflows.
type Plugin struct {
	githubClient *github.Client
}

// New creates a new Argo plugin instance.
func New() *Plugin {
	return &Plugin{
		githubClient: github.NewClient(),
	}
}

// NewWithClient creates a new Argo plugin with a custom GitHub client.
func NewWithClient(client *github.Client) *Plugin {
	return &Plugin{
		githubClient: client,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "argo"
}

// ListBinPaths returns the binary paths for Argo installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Argo execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version filenames for Argo.
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

// Uninstall removes an Argo installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Argo plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Argo Workflows CLI - The workflow engine for Kubernetes.
Argo is built from the official source archive using Go, which requires Go to be installed.`,
		Deps:   `Requires Go to be installed and available in PATH.`,
		Config: `No additional configuration required.`,
		Links: `Homepage: https://argo-workflows.readthedocs.io/
Source: https://github.com/argoproj/argo-workflows`,
	}
}

// ListAll lists all available Argo versions from GitHub tags.
func (p *Plugin) ListAll(ctx context.Context) ([]string, error) {
	tags, err := p.githubClient.GetTags(ctx, "https://github.com/argoproj/argo-workflows")
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "v3.") {
			versions = append(versions, strings.TrimPrefix(tag, "v"))
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return asdf.CompareVersions(versions[i], versions[j]) < 0
	})

	return versions, nil
}

// LatestStable returns the latest stable Argo version.
func (p *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := p.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errArgoNoVersionsFound
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
		return "", fmt.Errorf("%w: %s", errArgoNoVersionsMatching, query)
	}

	return filtered[len(filtered)-1], nil
}

// Download is a no-op for Argo since installation downloads the source archive directly.
func (*Plugin) Download(_ context.Context, _, _ string) error {
	return nil
}

// installBuildToolchains installs the Go and Node.js toolchains required to
// build the Argo UI into an asdf-style installs tree under ASDF_DATA_DIR or
// ~/.asdf. It reuses the shared Go and Node.js toolchain helpers so both
// toolchains are installed consistently.
func installBuildToolchains(ctx context.Context) error {
	if err := asdf_plugin_go.InstallGoToolchain(ctx); err != nil {
		return err
	}

	return asdf_plugin_nodejs.InstallNodeToolchain(ctx)
}

// Install method downloads the Argo Workflows source
// archive for the requested version and builds the argo CLI using go build.
func (*Plugin) Install(ctx context.Context, version, _, installPath string) error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go is required to install argo but was not found in PATH: %w", err)
	}

	binDir := filepath.Join(installPath, "bin")
	if err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "argo-src-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	archiveName := fmt.Sprintf("argo-workflows-%s.tar.gz", version)
	archivePath := filepath.Join(tempDir, archiveName)
	sourceDir := filepath.Join(tempDir, "argo-workflows-"+version)
	url := fmt.Sprintf("https://github.com/argoproj/argo-workflows/archive/refs/tags/v%s.tar.gz", version)

	asdf.Msgf("Downloading argo %s source from %s", version, url)

	if err := downloadFileFn(ctx, url, archivePath); err != nil {
		return fmt.Errorf("downloading argo source: %w", err)
	}

	tarCmd := execCommandContextFn(ctx, "tar", "-xzf", archivePath, "-C", tempDir)

	tarCmd.Stdout = os.Stderr

	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("extracting argo source: %w", err)
	}

	if err := ensureToolchainsFn(ctx, "golang", "nodejs"); err != nil {
		return err
	}

	npmPath := "npm"

	if asdfPath, lookErr := exec.LookPath("asdf"); lookErr == nil {
		whichCmd := exec.CommandContext(ctx, asdfPath, "which", "npm")

		whichCmd.Env = os.Environ()
		if out, whichErr := whichCmd.Output(); whichErr == nil {
			if resolved := strings.TrimSpace(string(out)); resolved != "" {
				npmPath = resolved
			}
		}
	}

	uiDir := filepath.Join(sourceDir, "ui")
	baseEnv := os.Environ()

	nodePath := "node"
	if asdfPath, lookErr := exec.LookPath("asdf"); lookErr == nil {
		whichNode := execCommandContextFn(ctx, asdfPath, "which", "node")

		whichNode.Env = baseEnv
		if out, whichErr := whichNode.Output(); whichErr == nil {
			if resolved := strings.TrimSpace(string(out)); resolved != "" {
				nodePath = resolved
			}
		}
	}

	nodeDir := filepath.Dir(nodePath)

	baseEnv = append(baseEnv, "PATH="+nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	uiToolVersions := filepath.Join(uiDir, ".tool-versions")
	if err := ensureToolVersionsFileFn(ctx, uiToolVersions, "nodejs", "golang"); err != nil {
		return err
	}

	yarnInstall := execCommandContextFn(ctx, npmPath, "exec", "yarn", "install")

	yarnInstall.Dir = uiDir
	yarnInstall.Stdout = os.Stderr
	yarnInstall.Stderr = os.Stderr

	yarnInstall.Env = baseEnv
	if err := yarnInstall.Run(); err != nil {
		return fmt.Errorf("installing argo UI dependencies with yarn: %w", err)
	}

	baseEnv = append(baseEnv,
		"NODE_ENV=production",
		"NODE_OPTIONS=--max-old-space-size=2048",
	)

	yarnBuild := execCommandContextFn(ctx, npmPath, "exec", "yarn", "build")

	yarnBuild.Dir = uiDir
	yarnBuild.Stdout = os.Stderr
	yarnBuild.Stderr = os.Stderr

	yarnBuild.Env = baseEnv
	if err := yarnBuild.Run(); err != nil {
		return fmt.Errorf("building argo UI with yarn build: %w", err)
	}

	buildCmd := execCommandContextFn(ctx, goPath, "build", "-o", filepath.Join(binDir, "argo"), "./cmd/argo")

	buildCmd.Dir = sourceDir
	buildCmd.Stdout = os.Stderr
	buildCmd.Stderr = os.Stderr
	buildCmd.Env = os.Environ()

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("building argo: %w", err)
	}

	binaryPath := filepath.Join(binDir, "argo")
	if _, err := statFn(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", errArgoBinaryNotFound, version)
	}

	fmt.Printf("Argo %s installed successfully\n", version)

	return nil
}
