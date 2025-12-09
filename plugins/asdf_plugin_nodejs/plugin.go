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

package asdf_plugin_nodejs

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

const (
	// nodeBuildGitURL is the node-build repository.
	nodeBuildGitURL = "https://github.com/nodenv/node-build.git"
	// nodeDistURL is the Node.js distribution URL.
	nodeDistURL = "https://nodejs.org/dist/"
	// nodeIndexURL is the Node.js version index.
	nodeIndexURL = "https://nodejs.org/dist/index.json"
	// nodeGitRepoURL is the Node.js GitHub repository for releases.
	nodeGitRepoURL = "https://github.com/nodejs/node"
)

// Plugin implements the asdf.Plugin interface for Node.js.
type Plugin struct {
	githubClient *github.Client
	nodeBuildDir string
	apiURL       string
	distURL      string
}

// New creates a new Node.js plugin instance.
func New() *Plugin {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Plugin{
		nodeBuildDir: filepath.Join(homeDir, ".asdf-node-build"),
		apiURL:       nodeIndexURL,
		distURL:      nodeDistURL,
		githubClient: github.NewClient(),
	}
}

// NewWithURLs creates a new Node.js plugin with custom URLs (for testing).
func NewWithURLs(apiURL, distURL string) *Plugin {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Plugin{
		nodeBuildDir: filepath.Join(homeDir, ".asdf-node-build"),
		apiURL:       apiURL,
		distURL:      distURL,
		githubClient: github.NewClient(),
	}
}

// NewWithBuildDir creates a new Node.js plugin with custom build directory (for testing).
func NewWithBuildDir(nodeBuildDir string) *Plugin {
	return &Plugin{
		nodeBuildDir: nodeBuildDir,
		apiURL:       nodeIndexURL,
		distURL:      nodeDistURL,
		githubClient: github.NewClient(),
	}
}

// newWithGitHubClient constructs a Plugin with a custom GitHub client. It is
// intended for tests that need to exercise ListAllFromGitHub without making
// real network requests.
func newWithGitHubClient(client *github.Client) *Plugin {
	return &Plugin{
		nodeBuildDir: ".asdf-node-build-test",
		apiURL:       nodeIndexURL,
		distURL:      nodeDistURL,
		githubClient: client,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "nodejs"
}

// ListBinPaths returns the binary paths for Node.js installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Node.js execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for Node.js.
func (*Plugin) ListLegacyFilenames() []string {
	return []string{".nvmrc", ".node-version"}
}

// ParseLegacyFile parses a legacy Node.js version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	version, err := asdf.ReadLegacyVersionFile(path)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(version, "lts") {
		return version, nil
	}

	return strings.TrimPrefix(version, "v"), nil
}

// Uninstall removes a Node.js installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Node.js plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Node.js - A JavaScript runtime built on Chrome's V8 JavaScript engine.
This plugin downloads pre-built Node.js binaries from https://nodejs.org/`,
		Deps: `No system dependencies required - uses pre-built binaries.`,
		Config: `Environment variables:
  ASDF_NODEJS_DEFAULT_PACKAGES_FILE - Path to default npm packages file (default: ~/.default-npm-packages)
  ASDF_NODEJS_AUTO_ENABLE_COREPACK - Enable corepack after install (default: false)
  NODEJS_CHECK_SIGNATURES - Verify GPG signatures (default: strict)
  NODEJS_ORG_MIRROR - Custom mirror URL for Node.js downloads`,
		Links: `Homepage: https://nodejs.org/
Documentation: https://nodejs.org/docs/
Downloads: https://nodejs.org/en/download/
Source: https://github.com/nodejs/node`,
	}
}
