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
	// errNodeChecksumNotFound is returned when the expected checksum entry cannot be found.
	errNodeChecksumNotFound = errors.New("checksum not found")
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

type (
	// NodeVersion represents a Node.js version from the API.
	NodeVersion struct {
		Version string `json:"version"`
		LTS     any    `json:"lts"`
		Date    string `json:"date"`
	}

	// NodejsPlugin implements the asdf.Plugin interface for Node.js.
	NodejsPlugin struct {
		githubClient *github.Client
		sourceBuild  *asdf.SourceBuildPlugin
		nodeBuildDir string
		apiURL       string
		distURL      string
	}
)

// NewNodejsPlugin creates a new Node.js plugin instance.
func NewNodejsPlugin() asdf.Plugin {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	plugin := &NodejsPlugin{
		nodeBuildDir: filepath.Join(homeDir, ".asdf-node-build"),
		apiURL:       nodeIndexURL,
		distURL:      nodeDistURL,
		githubClient: github.NewClient(),
	}

	createBinDir := false
	cfg := &asdf.SourceBuildPluginConfig{
		Name:                   "nodejs",
		BinDir:                 "bin",
		CreateBinDir:           &createBinDir,
		SkipDownload:           true,
		ArchiveType:            "tar.gz",
		ArchiveNameTemplate:    "node.tar.gz",
		AutoDetectExtractedDir: true,
		ExpectedArtifacts:      []string{"bin/node", "bin/npm", "bin/npx"},
		BuildVersion: func(_ context.Context, _, sourceDir, installPath string) error {
			return plugin.flattenExtractedDir(sourceDir, installPath)
		},
	}

	plugin.sourceBuild = asdf.NewSourceBuildPlugin(cfg)

	return plugin
}

// Name returns the plugin name.
func (*NodejsPlugin) Name() string {
	return "nodejs"
}

// ListBinPaths returns the binary paths for Node.js installations.
func (*NodejsPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Node.js execution.
func (*NodejsPlugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for Node.js.
func (*NodejsPlugin) ListLegacyFilenames() []string {
	return []string{".nvmrc", ".node-version"}
}

// ParseLegacyFile parses a legacy Node.js version file.
func (*NodejsPlugin) ParseLegacyFile(path string) (string, error) {
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
func (*NodejsPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Node.js plugin.
func (*NodejsPlugin) Help() asdf.PluginHelp {
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

// isLTS returns true if the LTS field indicates an LTS version.
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
func (plugin *NodejsPlugin) ListAll(ctx context.Context) ([]string, error) {
	if err := plugin.ensureNodeBuild(ctx); err == nil {
		versions, err := plugin.listAllFromNodeBuild(ctx)
		if err == nil {
			return versions, nil
		}
	}

	return plugin.listAllFromAPI(ctx)
}

// listAllFromNodeBuild lists versions using node-build definitions.
func (plugin *NodejsPlugin) listAllFromNodeBuild(ctx context.Context) ([]string, error) {
	nodeBuildPath := plugin.nodeBuildPath()
	cmd := exec.CommandContext(ctx, nodeBuildPath, "--definitions")

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
func (plugin *NodejsPlugin) listAllFromAPI(ctx context.Context) ([]string, error) {
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
func (plugin *NodejsPlugin) ListAllFromGitHub(ctx context.Context) ([]string, error) {
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
func (plugin *NodejsPlugin) nodeBuildPath() string {
	return filepath.Join(plugin.nodeBuildDir, "bin", "node-build")
}

// ensureNodeBuild ensures node-build is installed and up to date.
func (plugin *NodejsPlugin) ensureNodeBuild(ctx context.Context) error {
	return asdf.EnsureGitRepo(
		ctx,
		plugin.nodeBuildDir,
		nodeBuildGitURL,
		"Installing node-build...",
		"node-build installed successfully",
	)
}

// sortNodeVersions sorts Node.js versions in semver order.
func sortNodeVersions(versions []string) {
	asdf.SortVersions(versions)
}

// ResolveVersion resolves a version alias (like "lts") to an actual version.
func (plugin *NodejsPlugin) ResolveVersion(ctx context.Context, version string) (string, error) {
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
func (plugin *NodejsPlugin) getLatestLTS(ctx context.Context) (string, error) {
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
func (plugin *NodejsPlugin) getLTSByCodename(ctx context.Context, codename string) (string, error) {
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
func (plugin *NodejsPlugin) GetLTSCodenames(ctx context.Context) (map[string]string, error) {
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
func (plugin *NodejsPlugin) LatestStable(ctx context.Context, query string) (string, error) {
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

// Download downloads the specified Node.js version.
func (plugin *NodejsPlugin) Download(ctx context.Context, version, downloadPath string) error {
	platform, err := asdf.GetPlatform()
	if err != nil {
		return err
	}

	arch, err := getNodeArch()
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%sv%s/node-v%s-%s-%s.tar.gz", plugin.distURL, version, version, platform, arch)
	archivePath := filepath.Join(downloadPath, "node.tar.gz")

	asdf.Msgf("Downloading Node.js %s from %s", version, downloadURL)

	if err := asdf.DownloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("downloading Node.js %s: %w", version, err)
	}

	shasumsURL := fmt.Sprintf("%sv%s/SHASUMS256.txt", plugin.distURL, version)
	shasumsPath := filepath.Join(downloadPath, "SHASUMS256.txt")

	if err := asdf.DownloadFile(ctx, shasumsURL, shasumsPath); err != nil {
		asdf.Errf("Warning: could not download checksums: %v", err)
	} else {
		expectedFilename := fmt.Sprintf("node-v%s-%s-%s.tar.gz", version, platform, arch)
		if err := verifyNodeChecksum(archivePath, shasumsPath, expectedFilename); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}

		asdf.Msgf("Checksum verified")
	}

	return nil
}

// getNodeArch returns the architecture string for Node.js downloads.
func getNodeArch() (string, error) {
	arch, err := asdf.GetArch()
	if err != nil {
		return "", err
	}

	switch arch {
	case "amd64":
		return "x64", nil
	case "386":
		return "x86", nil
	case "arm64":
		return "arm64", nil
	case "armv6l":
		return "armv7l", nil
	default:
		return arch, nil
	}
}

// verifyNodeChecksum verifies the checksum of a Node.js download.
func verifyNodeChecksum(archivePath, shasumsPath, expectedFilename string) error {
	file, err := os.Open(shasumsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasSuffix(parts[1], expectedFilename) {
			return asdf.VerifySHA256(archivePath, parts[0])
		}
	}

	return fmt.Errorf("%w: %s", errNodeChecksumNotFound, expectedFilename)
}

// Install installs Node.js from the downloaded archive.
func (plugin *NodejsPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	if err := asdf.EnsureToolchains(ctx, "python"); err != nil {
		return err
	}

	archivePath := filepath.Join(downloadPath, "node.tar.gz")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		if err := plugin.Download(ctx, version, downloadPath); err != nil {
			return err
		}
	}

	asdf.Msgf("Installing Node.js %s to %s", version, installPath)

	if err := plugin.sourceBuild.Install(ctx, version, downloadPath, installPath); err != nil {
		return err
	}

	if downloadPath != "" {
		_ = os.RemoveAll(filepath.Join(downloadPath, "src"))
	}

	if err := plugin.installDefaultPackages(ctx, installPath); err != nil {
		asdf.Errf("Warning: failed to install default packages: %v", err)
	}

	if os.Getenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK") != "" {
		if err := plugin.enableCorepack(ctx, installPath); err != nil {
			asdf.Errf("Warning: failed to enable corepack: %v", err)
		}
	}

	asdf.Msgf("Node.js %s installed successfully", version)

	return nil
}

// flattenExtractedDir moves all contents from sourceDir to installPath.
func (*NodejsPlugin) flattenExtractedDir(sourceDir, installPath string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("reading extracted directory: %w", err)
	}

	for _, entry := range entries {
		src := filepath.Join(sourceDir, entry.Name())

		dst := filepath.Join(installPath, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("moving %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// installDefaultPackages installs packages from ~/.default-npm-packages.
func (*NodejsPlugin) installDefaultPackages(ctx context.Context, installPath string) error {
	defaultPkgsFile := os.Getenv("ASDF_NPM_DEFAULT_PACKAGES_FILE")
	if defaultPkgsFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil
		}

		defaultPkgsFile = filepath.Join(homeDir, ".default-npm-packages")
	}

	if _, err := os.Stat(defaultPkgsFile); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(defaultPkgsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	npmPath := filepath.Join(installPath, "bin", "npm")
	nodePath := filepath.Join(installPath, "bin", "node")

	var packages []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, " -") || strings.HasPrefix(line, "-") {
			args := strings.Fields(line)
			asdf.Msgf("Running: npm install -g %s", line)

			cmd := exec.CommandContext(ctx, npmPath, append([]string{"install", "-g"}, args...)...)

			cmd.Env = append(os.Environ(),
				"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
			)
			cmd.Stdout = os.Stderr

			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				asdf.Errf("Failed to install %s: %v", line, err)
			}

			continue
		}

		packages = append(packages, strings.Fields(line)...)
	}

	if len(packages) > 0 {
		asdf.Msgf("Running: npm install -g %s", strings.Join(packages, " "))

		cmd := exec.CommandContext(ctx, npmPath, append([]string{"install", "-g"}, packages...)...)

		cmd.Env = append(os.Environ(),
			"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
			"NODE="+nodePath,
		)
		cmd.Stdout = os.Stderr

		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			asdf.Errf("Failed to install packages: %v", err)
		}
	}

	return scanner.Err()
}

// enableCorepack enables corepack for this Node.js installation.
func (*NodejsPlugin) enableCorepack(ctx context.Context, installPath string) error {
	corepackPath := filepath.Join(installPath, "bin", "corepack")
	if _, err := os.Stat(corepackPath); os.IsNotExist(err) {
		return nil
	}

	asdf.Msgf("Enabling corepack...")

	cmd := exec.CommandContext(ctx, corepackPath, "enable")

	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// InstallNodeToolchain installs the Node.js toolchain into an asdf-style tree under
// ASDF_DATA_DIR (or $HOME/.asdf if unset) using the Node.js plugin implementation.
func InstallNodeToolchain(ctx context.Context) error {
	return asdf.InstallToolchain(ctx, "nodejs", NewNodejsPlugin())
}
