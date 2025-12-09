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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errNodeNoLTSVersionFound is returned when no LTS version exists in the Node.js index.
	errNodeNoLTSVersionFound = errors.New("no LTS version found")
	// errNodeLTSCodenameNotFound is returned when an LTS codename cannot be resolved.
	errNodeLTSCodenameNotFound = errors.New("LTS codename not found")
	// errNodeNoStableVersionFound is returned when no stable version matches a LatestStable query.
	errNodeNoStableVersionFound = errors.New("no stable version found for query")

	// execCommandContextFnNode wraps exec.CommandContext so tests can stub node-build
	// and git invocations without requiring the real binaries to be present.
	execCommandContextFnNode = exec.CommandContext //nolint:gochecknoglobals // configurable in tests
)

// NodeVersion represents a Node.js version from the API.
type NodeVersion struct {
	Version string `json:"version"`
	LTS     any    `json:"lts"`
	Date    string `json:"date"`
}

// isLTS returns true if the LTS field indicates an LTS version.
// LTS can be false (not LTS) or a string codename (LTS).
func isLTS(lts any) bool {
	if lts == nil {
		return false
	}

	if b, ok := lts.(bool); ok {
		return b
	}

	_, ok := lts.(string)

	return ok
}

// ListAll returns all available Node.js versions.
func (plugin *Plugin) ListAll(ctx context.Context) ([]string, error) {
	if err := plugin.ensureNodeBuild(ctx); err == nil {
		versions, err := plugin.listAllFromNodeBuild(ctx)
		if err == nil {
			return versions, nil
		}
	}

	return plugin.listAllFromAPI(ctx)
}

// listAllFromNodeBuild lists versions using node-build definitions.
func (plugin *Plugin) listAllFromNodeBuild(ctx context.Context) ([]string, error) {
	nodeBuildPath := plugin.nodeBuildPath()
	cmd := execCommandContextFnNode(ctx, nodeBuildPath, "--definitions")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing node versions: %w", err)
	}

	var versions []string

	re := regexp.MustCompile(`^[0-9.]+$`)
	for line := range strings.SplitSeq(string(output), "\n") {
		v := strings.TrimSpace(line)
		if re.MatchString(v) {
			versions = append(versions, v)
		}
	}

	sortNodeVersions(versions)

	return versions, nil
}

// listAllFromAPI lists versions from nodejs.org API.
func (plugin *Plugin) listAllFromAPI(ctx context.Context) ([]string, error) {
	content, err := asdf.DownloadString(ctx, plugin.apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching Node.js versions: %w", err)
	}

	var nodeVersions []NodeVersion
	if err := json.Unmarshal([]byte(content), &nodeVersions); err != nil {
		return nil, fmt.Errorf("parsing Node.js versions: %w", err)
	}

	versions := make([]string, 0, len(nodeVersions))
	for i := range nodeVersions {
		v := strings.TrimPrefix(nodeVersions[i].Version, "v")

		versions = append(versions, v)
	}

	sortNodeVersions(versions)

	return versions, nil
}

// ListAllFromGitHub returns all available Node.js versions from GitHub releases.
func (plugin *Plugin) ListAllFromGitHub(ctx context.Context) ([]string, error) {
	client := plugin.githubClient
	if client == nil {
		client = github.NewClient()
	}

	releases, err := client.GetReleases(ctx, nodeGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("fetching Node.js releases: %w", err)
	}

	re := regexp.MustCompile(`^v(\d+\.\d+\.\d+)$`)

	var versions []string

	for _, tag := range releases {
		if matches := re.FindStringSubmatch(tag); len(matches) == 2 {
			versions = append(versions, matches[1])
		}
	}

	sortNodeVersions(versions)

	return versions, nil
}

// nodeBuildPath returns the path to node-build executable.
func (plugin *Plugin) nodeBuildPath() string {
	return filepath.Join(plugin.nodeBuildDir, "bin", "node-build")
}

// ensureNodeBuild ensures node-build is installed and up to date.
func (plugin *Plugin) ensureNodeBuild(ctx context.Context) error {
	nodeBuildPath := plugin.nodeBuildPath()

	if _, err := os.Stat(nodeBuildPath); os.IsNotExist(err) {
		asdf.Msgf("Installing node-build...")

		if err := os.MkdirAll(filepath.Dir(plugin.nodeBuildDir), asdf.CommonDirectoryPermission); err != nil {
			return fmt.Errorf("creating node-build directory: %w", err)
		}

		cmd := execCommandContextFnNode(ctx, "git", "clone", "--depth", "1", nodeBuildGitURL, plugin.nodeBuildDir)

		cmd.Stdout = os.Stderr

		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cloning node-build: %w", err)
		}

		asdf.Msgf("node-build installed successfully")
	} else {
		cmd := execCommandContextFnNode(ctx, "git", "-C", plugin.nodeBuildDir, "pull", "--ff-only")

		_ = cmd.Run() //nolint:errcheck // Ignore errors on update
	}

	return nil
}

// sortNodeVersions sorts Node.js versions in semver order.
func sortNodeVersions(versions []string) {
	asdf.SortVersions(versions)
}

// ResolveVersion resolves a version alias (like "lts") to an actual version.
func (plugin *Plugin) ResolveVersion(ctx context.Context, version string) (string, error) {
	switch strings.ToLower(version) {
	case "lts", "lts/*":
		return plugin.getLatestLTS(ctx)
	}

	if after, ok := strings.CutPrefix(strings.ToLower(version), "lts/"); ok {
		codename := after
		return plugin.getLTSByCodename(ctx, codename)
	}

	return version, nil
}

// getLatestLTS returns the latest LTS version.
func (plugin *Plugin) getLatestLTS(ctx context.Context) (string, error) {
	content, err := asdf.DownloadString(ctx, plugin.apiURL)
	if err != nil {
		return "", err
	}

	var nodeVersions []NodeVersion
	if err := json.Unmarshal([]byte(content), &nodeVersions); err != nil {
		return "", err
	}

	for i := range nodeVersions {
		if isLTS(nodeVersions[i].LTS) {
			return strings.TrimPrefix(nodeVersions[i].Version, "v"), nil
		}
	}

	return "", errNodeNoLTSVersionFound
}

// getLTSByCodename returns the latest version for an LTS codename.
func (plugin *Plugin) getLTSByCodename(ctx context.Context, codename string) (string, error) {
	content, err := asdf.DownloadString(ctx, plugin.apiURL)
	if err != nil {
		return "", err
	}

	var nodeVersions []NodeVersion
	if err := json.Unmarshal([]byte(content), &nodeVersions); err != nil {
		return "", err
	}

	for i := range nodeVersions {
		lts, ok := nodeVersions[i].LTS.(string)
		if !ok {
			continue
		}

		if strings.EqualFold(lts, codename) {
			return strings.TrimPrefix(nodeVersions[i].Version, "v"), nil
		}
	}

	return "", fmt.Errorf("%w: %s", errNodeLTSCodenameNotFound, codename)
}

// GetLTSCodenames returns all available LTS codenames.
func (plugin *Plugin) GetLTSCodenames(ctx context.Context) (map[string]string, error) {
	content, err := asdf.DownloadString(ctx, plugin.apiURL)
	if err != nil {
		return nil, err
	}

	var nodeVersions []NodeVersion
	if err := json.Unmarshal([]byte(content), &nodeVersions); err != nil {
		return nil, err
	}

	codenames := make(map[string]string)
	for i := range nodeVersions {
		lts, ok := nodeVersions[i].LTS.(string)
		if !ok {
			continue
		}

		if _, exists := codenames[lts]; !exists {
			codenames[lts] = strings.TrimPrefix(nodeVersions[i].Version, "v")
		}
	}

	return codenames, nil
}

// LatestStable returns the latest stable Node.js version.
func (plugin *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	content, err := asdf.DownloadString(ctx, plugin.apiURL)
	if err != nil {
		return "", err
	}

	var nodeVersions []NodeVersion
	if err := json.Unmarshal([]byte(content), &nodeVersions); err != nil {
		return "", err
	}

	if query == "lts" || query == "lts/*" {
		for i := range nodeVersions {
			if _, ok := nodeVersions[i].LTS.(string); ok {
				return strings.TrimPrefix(nodeVersions[i].Version, "v"), nil
			}
		}
	}

	if after, ok := strings.CutPrefix(query, "lts/"); ok {
		codename := after

		return plugin.getLTSByCodename(ctx, codename)
	}

	for i := range nodeVersions {
		version := strings.TrimPrefix(nodeVersions[i].Version, "v")
		if query != "" && !strings.HasPrefix(version, query) {
			continue
		}

		parts := strings.Split(version, ".")
		if len(parts) == 0 {
			continue
		}

		major, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		if major%2 == 0 || isLTS(nodeVersions[i].LTS) {
			return version, nil
		}
	}

	for i := range nodeVersions {
		version := strings.TrimPrefix(nodeVersions[i].Version, "v")
		if query == "" || strings.HasPrefix(version, query) {
			return version, nil
		}
	}

	return "", fmt.Errorf("%w: %s", errNodeNoStableVersionFound, query)
}
