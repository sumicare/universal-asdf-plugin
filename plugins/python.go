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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

const (
	// pyenvGitURL is the pyenv repository for python-build.
	pyenvGitURL = "https://github.com/pyenv/pyenv.git"
	// pythonFTPURL is the Python FTP server for version listing.
	pythonFTPURL = "https://www.python.org/ftp/python/"
)

var (
	// errPythonNoVersionsFound is returned when no Python versions are discovered.
	errPythonNoVersionsFound = errors.New("no versions found")
	// stableCPythonVersionRE matches stable CPython versions in plain X.Y.Z form.
	stableCPythonVersionRE = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// PythonPlugin implements the asdf.Plugin interface for Python.
type PythonPlugin struct {
	*asdf.SourceBuildPlugin

	pyenvDir string
}

// NewPythonPlugin creates a new Python plugin instance.
func NewPythonPlugin() asdf.Plugin {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	plugin := &PythonPlugin{
		pyenvDir: filepath.Join(homeDir, ".asdf-python-build"),
	}

	plugin.SourceBuildPlugin = asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name:              "python",
		RepoOwner:         "python",
		RepoName:          "cpython",
		VersionPrefix:     "v",
		VersionFilter:     `^\d+\.\d+\.\d+$`,
		BinDir:            "bin",
		SkipDownload:      true,
		SkipExtract:       true,
		LegacyFilenames:   []string{".python-version"},
		ExpectedArtifacts: []string{"bin/python"},

		BuildVersion: func(ctx context.Context, version, _, installPath string) error {
			// Check build dependencies
			for _, dep := range []string{"make", "gcc"} {
				if _, err := exec.LookPath(dep); err != nil {
					asdf.Msgf("Warning: Build dependency %s not found. Installation may fail.", dep)
				}
			}

			pythonBuildPath := filepath.Join(
				plugin.pyenvDir,
				"plugins",
				"python-build",
				"bin",
				"python-build",
			)

			asdf.Msgf("Installing Python %s to %s", version, installPath)

			// Handle patches
			patchURL := os.Getenv("ASDF_PYTHON_PATCH_URL")
			patchDir := os.Getenv("ASDF_PYTHON_PATCHES_DIRECTORY")

			var cmd *exec.Cmd

			if patchURL != "" {
				asdf.Msgf("Applying patch from %s", patchURL)

				patchData, err := exec.CommandContext(ctx, "curl", "-sSL", patchURL).Output()
				if err != nil {
					return fmt.Errorf("downloading patch from %s: %w", patchURL, err)
				}

				cmd = exec.CommandContext(ctx, pythonBuildPath, "--patch", version, installPath)
				cmd.Stdin = bytes.NewReader(patchData)
			} else if patchDir != "" {
				patchFile := filepath.Join(patchDir, version+".patch")
				if _, err := os.Stat(patchFile); err == nil {
					asdf.Msgf("Applying patch from %s", patchFile)

					patchReader, err := os.Open(patchFile)
					if err != nil {
						return fmt.Errorf("opening patch file %s: %w", patchFile, err)
					}
					defer patchReader.Close()

					cmd = exec.CommandContext(ctx, pythonBuildPath, version, installPath, "-p")
					cmd.Stdin = patchReader
				}
			}

			if cmd == nil {
				cmd = exec.CommandContext(ctx, pythonBuildPath, version, installPath)
			}

			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			cmd.Env = os.Environ()

			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("running python-build: %w", err)
			}

			return nil
		},

		PostInstallVersion: func(ctx context.Context, _ /* version */, installPath string) error {
			// Install default packages
			if err := installDefaultPackages(ctx, installPath); err != nil {
				return err
			}

			// Reshim - ensure all binaries are executable
			binDir := filepath.Join(installPath, "bin")

			entries, err := os.ReadDir(binDir)
			if err != nil {
				return fmt.Errorf("reading bin directory: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				filePath := filepath.Join(binDir, entry.Name())

				err := os.Chmod(filePath, asdf.CommonExecutablePermission)
				if err != nil {
					asdf.Msgf("Warning: Failed to chmod %s: %v", filePath, err)
				}
			}

			return nil
		},
	})

	return plugin
}

// Name returns the plugin name.
func (*PythonPlugin) Name() string {
	return "python"
}

// ListBinPaths returns the binary paths for Python installations.
func (*PythonPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Python execution.
func (*PythonPlugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for Python.
func (*PythonPlugin) ListLegacyFilenames() []string {
	return []string{".python-version"}
}

// ParseLegacyFile parses a legacy Python version file.
func (*PythonPlugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes a Python installation.
func (*PythonPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Python plugin.
func (*PythonPlugin) Help() asdf.PluginHelp {
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

// ListAll returns all available Python versions using python-build definitions.
func (plugin *PythonPlugin) ListAll(ctx context.Context) ([]string, error) {
	// Ensure python-build is available
	if err := plugin.Download(ctx, "", ""); err != nil {
		return nil, err
	}

	pythonBuildPath := filepath.Join(
		plugin.pyenvDir,
		"plugins",
		"python-build",
		"bin",
		"python-build",
	)
	cmd := exec.CommandContext(ctx, pythonBuildPath, "--definitions")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing python versions: %w", err)
	}

	versions := strings.Fields(string(output))
	asdf.SortVersions(versions)

	stable := asdf.FilterVersions(versions, func(v string) bool {
		if !stableCPythonVersionRE.MatchString(v) {
			return false
		}

		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		return stable, nil
	}

	return versions, nil
}

// LatestStable returns the latest stable Python version.
func (*PythonPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	// Fetch versions from FTP
	content, err := asdf.DownloadString(ctx, pythonFTPURL)
	if err != nil {
		return "", fmt.Errorf("fetching Python versions: %w", err)
	}

	re := regexp.MustCompile(`href="(\d+\.\d+\.\d+)/"`)
	matches := re.FindAllStringSubmatch(content, -1)

	var versions []string

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 && !seen[match[1]] {
			seen[match[1]] = true
			versions = append(versions, match[1])
		}
	}

	asdf.SortVersions(versions)

	stable := asdf.FilterVersions(versions, func(v string) bool {
		if !stableCPythonVersionRE.MatchString(v) {
			return false
		}

		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		versions = stable
	}

	if len(versions) == 0 {
		return "", errPythonNoVersionsFound
	}

	if query != "" {
		filtered := asdf.FilterVersions(versions, func(v string) bool {
			return strings.HasPrefix(v, query)
		})
		if len(filtered) > 0 {
			versions = filtered
		}
	}

	stableVersions := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stableVersions) == 0 {
		return versions[len(versions)-1], nil
	}

	return stableVersions[len(stableVersions)-1], nil
}

// Download ensures python-build tooling is installed.
func (plugin *PythonPlugin) Download(ctx context.Context, _, _ string) error {
	return asdf.EnsureGitRepo(
		ctx,
		plugin.pyenvDir,
		pyenvGitURL,
		"Installing python-build from pyenv...",
		"python-build installed successfully",
	)
}

// Install installs the specified Python version using python-build.
func (plugin *PythonPlugin) Install(ctx context.Context, version, _, installPath string) error {
	err := plugin.SourceBuildPlugin.Install(ctx, version, "", installPath)
	if err != nil {
		return err
	}

	asdf.Msgf("Python %s installed successfully", version)

	return nil
}

// installDefaultPackages installs default pip packages if the configuration file exists.
func installDefaultPackages(ctx context.Context, installPath string) error {
	defaultPackagesFile := os.Getenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
	if defaultPackagesFile == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultPackagesFile = filepath.Join(homeDir, ".default-python-packages")
		}
	}

	if defaultPackagesFile == "" {
		return nil
	}

	if _, err := os.Stat(defaultPackagesFile); err != nil {
		return nil
	}

	asdf.Msgf("Installing default pip packages from %s", defaultPackagesFile)

	file, err := os.Open(defaultPackagesFile)
	if err != nil {
		return fmt.Errorf("opening default packages file: %w", err)
	}
	defer file.Close()

	pipPath := filepath.Join(installPath, "bin", "pip")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pkg := strings.TrimSpace(scanner.Text())
		if pkg == "" || strings.HasPrefix(pkg, "#") {
			continue
		}

		asdf.Msgf("Installing %s", pkg)

		pipCmd := exec.CommandContext(ctx, pipPath, "install", pkg)

		pipCmd.Stdout = os.Stderr
		pipCmd.Stderr = os.Stderr

		err := pipCmd.Run()
		if err != nil {
			asdf.Msgf("Warning: Failed to install %s: %v", pkg, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading default packages file: %w", err)
	}

	return nil
}
