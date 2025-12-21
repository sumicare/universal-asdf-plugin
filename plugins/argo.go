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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// errArgoNoVersionsFound is returned when no Argo versions are discovered.
	errArgoNoVersionsFound = errors.New("no versions found")
	// errArgoNoVersionsMatching is returned when no versions match a LatestStable query.
	errArgoNoVersionsMatching = errors.New("no versions matching query")
	// errArgoBinaryNotFound is returned when the installed argo binary cannot be located.
	errArgoBinaryNotFound = errors.New("argo binary not found after installation")
)

// ArgoPlugin implements the asdf.Plugin interface for Argo Workflows.
type ArgoPlugin struct {
	*asdf.SourceBuildPlugin
}

// NewArgoPlugin creates a new Argo plugin instance.
//
//nolint:ireturn // returns asdf.Plugin interface implementation
func NewArgoPlugin() asdf.Plugin {
	useTags := true

	return &ArgoPlugin{asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name:          "argo",
		RepoOwner:     "argoproj",
		RepoName:      "argo-workflows",
		VersionPrefix: "v",
		UseTags:       useTags,
		VersionFilter: `^3\.`,

		Help: asdf.PluginHelp{
			Overview: `Argo Workflows CLI - The workflow engine for Kubernetes.
Argo is built from the official source archive using Go, which requires Go to be installed.`,
			Deps:   `Requires Go to be installed and available in PATH.`,
			Config: `No additional configuration required.`,
			Links: `Homepage: https://argo-workflows.readthedocs.io/
Source: https://github.com/argoproj/argo-workflows`,
		},

		BuildVersion: func(ctx context.Context, _ /* version */, sourceDir, installPath string) error {
			goPath, err := exec.LookPath("go")
			if err != nil {
				return fmt.Errorf(
					"go is required to install argo but was not found in PATH: %w",
					err,
				)
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
				whichNode := exec.CommandContext(ctx, asdfPath, "which", "node")

				whichNode.Env = baseEnv
				if out, whichErr := whichNode.Output(); whichErr == nil {
					if resolved := strings.TrimSpace(string(out)); resolved != "" {
						nodePath = resolved
					}
				}
			}

			nodeDir := filepath.Dir(nodePath)

			baseEnv = append(baseEnv,
				"PATH="+nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"CI=true", // Ensure non-interactive mode for build tools
			)

			// Ensure source directory has .tool-versions with golang
			sourceToolVersions := filepath.Join(sourceDir, ".tool-versions")
			if err := asdf.EnsureToolVersionsFile(ctx, sourceToolVersions, "golang"); err != nil {
				return err
			}

			uiToolVersions := filepath.Join(uiDir, ".tool-versions")
			if err := asdf.EnsureToolVersionsFile(ctx, uiToolVersions, "nodejs", "golang"); err != nil {
				return err
			}

			// Install global webpack as requested to avoid issues with yarn install/build
			// We install webpack-cli as well since it's often required.
			installWebpack := exec.CommandContext(
				ctx,
				npmPath,
				"install",
				"-g",
				"webpack",
				"webpack-cli",
			)

			installWebpack.Dir = uiDir // Run in UI dir context, though global install shouldn't strictly require it
			installWebpack.Stdout = os.Stderr
			installWebpack.Stderr = os.Stderr

			installWebpack.Env = baseEnv
			if err := installWebpack.Run(); err != nil {
				return fmt.Errorf("installing global webpack: %w", err)
			}

			yarnInstall := exec.CommandContext(ctx, npmPath, "exec", "yarn", "install")

			yarnInstall.Dir = uiDir
			yarnInstall.Stdout = os.Stderr
			yarnInstall.Stderr = os.Stderr

			yarnInstall.Env = baseEnv
			if err := yarnInstall.Run(); err != nil {
				return fmt.Errorf("installing argo UI dependencies with yarn: %w", err)
			}

			// Add node_modules/.bin to PATH for webpack and other build tools
			nodeModulesBin := filepath.Join(uiDir, "node_modules", ".bin")

			// Construct PATH that includes:
			// 1. local node_modules/.bin (for project-specific tools)
			// 2. nodeDir (for global webpack/webpack-cli and node/npm)
			// 3. original PATH (for system tools)
			buildPath := nodeModulesBin + string(os.PathListSeparator) +
				nodeDir + string(os.PathListSeparator) +
				os.Getenv("PATH")

			baseEnv = append(baseEnv,
				"PATH="+buildPath,
				"NODE_ENV=production",
				"NODE_OPTIONS=--max-old-space-size=2048",
			)

			yarnBuild := exec.CommandContext(ctx, npmPath, "exec", "yarn", "build")

			yarnBuild.Dir = uiDir
			yarnBuild.Stdout = os.Stderr
			yarnBuild.Stderr = os.Stderr

			yarnBuild.Env = baseEnv
			if err := yarnBuild.Run(); err != nil {
				return fmt.Errorf("building argo UI with yarn build: %w", err)
			}

			binDir := filepath.Join(installPath, "bin")
			if err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission); err != nil {
				return fmt.Errorf("creating bin directory: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Building argo binary in %s from %s\n", binDir, sourceDir)

			buildCmd := exec.CommandContext(
				ctx,
				goPath,
				"build",
				"-o",
				filepath.Join(binDir, "argo"),
				"./cmd/argo",
			)

			buildCmd.Dir = sourceDir
			buildCmd.Stdout = os.Stderr
			buildCmd.Stderr = os.Stderr
			buildCmd.Env = os.Environ()

			if err := buildCmd.Run(); err != nil {
				return fmt.Errorf("building argo: %w", err)
			}

			// Verify the binary was created
			binaryPath := filepath.Join(binDir, "argo")
			if _, err := os.Stat(binaryPath); err != nil {
				return fmt.Errorf("argo binary not found after build at %s: %w", binaryPath, err)
			}

			fmt.Fprintf(os.Stderr, "Argo binary created at %s\n", binaryPath)

			return nil
		},

		PostInstallVersion: func(_ context.Context, _ /* version */, installPath string) error {
			binaryPath := filepath.Join(installPath, "bin", "argo")
			if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
				return errArgoBinaryNotFound
			}

			return nil
		},

		ExpectedArtifacts: []string{"bin/argo"},
	})}
}

// Name returns the plugin name.
func (*ArgoPlugin) Name() string {
	return "argo"
}

// Dependencies returns the list of plugins that must be installed before Argo.
func (*ArgoPlugin) Dependencies() []string {
	return []string{"golang", "nodejs"}
}

// ListBinPaths returns the binary paths for Argo installations.
func (*ArgoPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Argo execution.
func (*ArgoPlugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version filenames for Argo.
func (*ArgoPlugin) ListLegacyFilenames() []string {
	return make([]string, 0)
}

// ParseLegacyFile parses a legacy version file.
func (*ArgoPlugin) ParseLegacyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// Uninstall removes an Argo installation.
func (*ArgoPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Argo plugin.
func (plugin *ArgoPlugin) Help() asdf.PluginHelp {
	return plugin.SourceBuildPlugin.Help()
}

// ListAll lists all available Argo versions from GitHub tags.
func (plugin *ArgoPlugin) ListAll(ctx context.Context) ([]string, error) {
	return plugin.SourceBuildPlugin.ListAll(ctx)
}

// LatestStable returns the latest stable Argo version.
func (plugin *ArgoPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errArgoNoVersionsFound
	}

	latest := asdf.LatestVersion(versions, query)
	if latest == "" {
		return "", fmt.Errorf("%w: %s", errArgoNoVersionsMatching, query)
	}

	return latest, nil
}

// Download is a no-op for Argo since installation downloads the source archive directly.
func (*ArgoPlugin) Download(_ context.Context, _, _ string) error {
	return nil
}

// Install method downloads the Argo Workflows source
// archive for the requested version and builds the argo CLI using go build.
func (plugin *ArgoPlugin) Install(
	ctx context.Context,
	version, downloadPath, installPath string,
) error {
	err := plugin.SourceBuildPlugin.Install(ctx, version, downloadPath, installPath)
	if err != nil {
		if errors.Is(err, errArgoBinaryNotFound) {
			return fmt.Errorf("%w: %s", errArgoBinaryNotFound, version)
		}

		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Argo %s installed successfully\n", version)

	return nil
}
