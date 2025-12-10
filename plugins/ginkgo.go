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
	// errGinkgoNoVersionsFound is returned when no Ginkgo versions are discovered.
	errGinkgoNoVersionsFound = errors.New("no versions found")
	// errGinkgoNoVersionsMatching is returned when no versions match a LatestStable query.
	errGinkgoNoVersionsMatching = errors.New("no versions matching query")
	// errGinkgoBinaryNotFound is returned when the installed ginkgo binary cannot be located.
	errGinkgoBinaryNotFound = errors.New("ginkgo binary not found after installation")
)

// GinkgoPlugin implements the asdf.Plugin interface for Ginkgo.
type GinkgoPlugin struct {
	*asdf.SourceBuildPlugin
}

// NewGinkgoPlugin creates a new Ginkgo plugin instance.
func NewGinkgoPlugin() asdf.Plugin {
	return &GinkgoPlugin{asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name:          "ginkgo",
		RepoOwner:     "onsi",
		RepoName:      "ginkgo",
		VersionPrefix: "v",
		VersionFilter: `^2\.`,

		Help: asdf.PluginHelp{
			Overview: `Ginkgo - A BDD-style Go testing framework.
Ginkgo is built from the official source archive using Go, which requires Go to be installed.`,
			Deps:   `Requires Go to be installed and available in PATH.`,
			Config: `No additional configuration required.`,
			Links: `Homepage: https://onsi.github.io/ginkgo/
Source: https://github.com/onsi/ginkgo`,
		},

		BuildVersion: func(ctx context.Context, _ /* version */, sourceDir, installPath string) error {
			goPath, err := exec.LookPath("go")
			if err != nil {
				return fmt.Errorf("go is required to install ginkgo but was not found in PATH: %w", err)
			}

			binDir := filepath.Join(installPath, "bin")
			dest := filepath.Join(binDir, "ginkgo")

			buildCmd := exec.CommandContext(ctx, goPath, "build", "-o", dest, "./ginkgo")

			buildCmd.Dir = sourceDir
			buildCmd.Stdout = os.Stderr
			buildCmd.Stderr = os.Stderr
			buildCmd.Env = os.Environ()

			if err := buildCmd.Run(); err != nil {
				return fmt.Errorf("building ginkgo: %w", err)
			}

			return nil
		},

		PostInstallVersion: func(_ context.Context, _ /* version */, installPath string) error {
			binaryPath := filepath.Join(installPath, "bin", "ginkgo")
			if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
				return errGinkgoBinaryNotFound
			}

			return nil
		},

		ExpectedArtifacts: []string{"bin/ginkgo"},
	})}
}

// Name returns the plugin name.
func (*GinkgoPlugin) Name() string {
	return "ginkgo"
}

// ListBinPaths returns the binary paths for Ginkgo installations.
func (*GinkgoPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Ginkgo execution.
func (*GinkgoPlugin) ExecEnv(_ string) map[string]string {
	return make(map[string]string)
}

// ListLegacyFilenames returns legacy version filenames for Ginkgo.
func (*GinkgoPlugin) ListLegacyFilenames() []string {
	return make([]string, 0)
}

// ParseLegacyFile parses a legacy version file.
func (*GinkgoPlugin) ParseLegacyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// Uninstall removes a Ginkgo installation.
func (*GinkgoPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Ginkgo plugin.
func (plugin *GinkgoPlugin) Help() asdf.PluginHelp {
	return plugin.SourceBuildPlugin.Help()
}

// ListAll lists all available Ginkgo versions from GitHub tags.
func (plugin *GinkgoPlugin) ListAll(ctx context.Context) ([]string, error) {
	return plugin.SourceBuildPlugin.ListAll(ctx)
}

// LatestStable returns the latest stable Ginkgo version.
func (plugin *GinkgoPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errGinkgoNoVersionsFound
	}

	latest := asdf.LatestVersion(versions, query)
	if latest == "" {
		return "", fmt.Errorf("%w: %s", errGinkgoNoVersionsMatching, query)
	}

	return latest, nil
}

// Download is a no-op for Ginkgo since installation downloads the source archive directly.
func (*GinkgoPlugin) Download(_ context.Context, _, _ string) error {
	return nil
}

// Install downloads the Ginkgo source archive and builds the ginkgo CLI using go build.
func (plugin *GinkgoPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	if err := plugin.SourceBuildPlugin.Install(ctx, version, downloadPath, installPath); err != nil {
		if errors.Is(err, errGinkgoBinaryNotFound) {
			return fmt.Errorf("%w: %s", errGinkgoBinaryNotFound, version)
		}

		return err
	}

	fmt.Printf("Ginkgo %s installed successfully\n", version)

	return nil
}
