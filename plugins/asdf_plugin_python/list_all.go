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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// errPythonNoVersionsFound is returned when no Python versions are discovered.
var errPythonNoVersionsFound = errors.New("no versions found")

// ListAll returns all available Python versions using python-build definitions.
func (plugin *Plugin) ListAll(ctx context.Context) ([]string, error) {
	if err := plugin.ensurePythonBuild(ctx); err != nil {
		return nil, fmt.Errorf("ensuring python-build: %w", err)
	}

	pythonBuildPath := plugin.pythonBuildPath()

	cmd := exec.CommandContext(ctx, pythonBuildPath, "--definitions")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing python versions: %w", err)
	}

	versions := strings.Fields(string(output))
	sortPythonVersions(versions)

	return versions, nil
}

// pythonBuildPath returns the path to python-build executable.
func (plugin *Plugin) pythonBuildPath() string {
	return filepath.Join(plugin.pyenvDir, "plugins", "python-build", "bin", "python-build")
}

// ensurePythonBuild ensures python-build is installed and up to date.
func (plugin *Plugin) ensurePythonBuild(ctx context.Context) error {
	pythonBuildPath := plugin.pythonBuildPath()

	if _, err := os.Stat(pythonBuildPath); os.IsNotExist(err) {
		asdf.Msgf("Installing python-build from pyenv...")

		if err := os.MkdirAll(filepath.Dir(plugin.pyenvDir), asdf.CommonDirectoryPermission); err != nil {
			return fmt.Errorf("creating pyenv directory: %w", err)
		}

		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", pyenvGitURL, plugin.pyenvDir) //nolint:gosec // G204: no shell, fixed arguments

		cmd.Stdout = os.Stderr

		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cloning pyenv: %w", err)
		}

		asdf.Msgf("python-build installed successfully")
	} else {
		cmd := exec.CommandContext(ctx, "git", "-C", plugin.pyenvDir, "pull", "--ff-only") //nolint:gosec // G204: no shell, fixed arguments

		_ = cmd.Run() //nolint:errcheck // Ignore errors on update
	}

	return nil
}

// sortPythonVersions sorts Python versions in semver order.
func sortPythonVersions(versions []string) {
	asdf.SortVersions(versions)
}

// ListAllFromFTP returns all available Python versions from python.org FTP.
// This is an alternative method that doesn't require python-build.
func (plugin *Plugin) ListAllFromFTP(ctx context.Context) ([]string, error) {
	content, err := asdf.DownloadString(ctx, plugin.ftpURL)
	if err != nil {
		return nil, fmt.Errorf("fetching Python versions: %w", err)
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

	sortPythonVersions(versions)

	return versions, nil
}

// ListAllFromGitHub returns all available Python versions from GitHub tags.
func (plugin *Plugin) ListAllFromGitHub(ctx context.Context) ([]string, error) {
	tags, err := plugin.githubClient.GetTags(ctx, pythonGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("fetching Python tags: %w", err)
	}

	re := regexp.MustCompile(`^v(\d+\.\d+\.\d+)$`)

	var versions []string

	for _, tag := range tags {
		if matches := re.FindStringSubmatch(tag); len(matches) == 2 {
			versions = append(versions, matches[1])
		}
	}

	sortPythonVersions(versions)

	return versions, nil
}

// LatestStable returns the latest stable Python version.
func (plugin *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAllFromFTP(ctx)
	if err != nil {
		return "", err
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

	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !strings.Contains(v, "a") && !strings.Contains(v, "b") &&
			!strings.Contains(v, "rc") && !strings.Contains(v, "alpha") &&
			!strings.Contains(v, "beta")
	})

	if len(stable) == 0 {
		return versions[len(versions)-1], nil
	}

	return stable[len(stable)-1], nil
}
