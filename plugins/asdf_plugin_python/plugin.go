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

package asdf_plugin_python

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

const (
	// pyenvGitURL is the pyenv repository for python-build.
	pyenvGitURL = "https://github.com/pyenv/pyenv.git"
	// pythonFTPURL is the Python FTP server for version listing.
	pythonFTPURL = "https://www.python.org/ftp/python/"
	// pythonGitRepoURL is the CPython GitHub repository for version tags.
	pythonGitRepoURL = "https://github.com/python/cpython"
)

// Plugin implements the asdf.Plugin interface for Python.
type Plugin struct {
	githubClient *github.Client
	pyenvDir     string
	ftpURL       string
}

// New creates a new Python plugin instance.
func New() *Plugin {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Plugin{
		pyenvDir:     filepath.Join(homeDir, ".asdf-python-build"),
		ftpURL:       pythonFTPURL,
		githubClient: github.NewClient(),
	}
}

// NewWithURLs creates a new Python plugin with custom URLs (for testing).
func NewWithURLs(ftpURL string, githubClient *github.Client) *Plugin {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Plugin{
		pyenvDir:     filepath.Join(homeDir, ".asdf-python-build"),
		ftpURL:       ftpURL,
		githubClient: githubClient,
	}
}

// NewWithBuildDir creates a new Python plugin with custom build directory (for testing).
func NewWithBuildDir(pyenvDir string) *Plugin {
	return &Plugin{
		pyenvDir:     pyenvDir,
		ftpURL:       pythonFTPURL,
		githubClient: github.NewClient(),
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "python"
}

// ListBinPaths returns the binary paths for Python installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Python execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for Python.
func (*Plugin) ListLegacyFilenames() []string {
	return []string{".python-version"}
}

// ParseLegacyFile parses a legacy Python version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes a Python installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Python plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Python - A programming language that lets you work quickly and integrate systems effectively.
This plugin uses python-build (from pyenv) to compile Python from source.`,
		Deps: `Build dependencies (Debian/Ubuntu):
  build-essential libssl-dev zlib1g-dev libbz2-dev libreadline-dev
  libsqlite3-dev curl libncursesw5-dev xz-utils tk-dev libxml2-dev
  libxmlsec1-dev libffi-dev liblzma-dev`,
		Config: `Environment variables:
  ASDF_PYTHON_DEFAULT_PACKAGES_FILE - Path to default pip packages file (default: ~/.default-python-packages)
  ASDF_PYTHON_PATCH_URL - URL to patch file to apply during build
  ASDF_PYTHON_PATCHES_DIRECTORY - Directory containing patch files
  PYTHON_BUILD_MIRROR_URL - Custom mirror URL for Python source downloads`,
		Links: `Homepage: https://www.python.org/
Documentation: https://docs.python.org/
Downloads: https://www.python.org/downloads/
Source: https://github.com/python/cpython`,
	}
}
