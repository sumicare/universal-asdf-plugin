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

package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/plugins"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_asdf"
)

var (
	// errPluginNameRequired is returned when no plugin name can be determined.
	errPluginNameRequired = errors.New("plugin name required. Set ASDF_PLUGIN_NAME or specify as argument")
	// errUnknownCommand is returned for unsupported top-level commands.
	errUnknownCommand = errors.New("unknown command")
	// errASDFInstallVersionNotSet is returned when ASDF_INSTALL_VERSION is missing.
	errASDFInstallVersionNotSet = errors.New("ASDF_INSTALL_VERSION not set")
	// errASDFDownloadPathNotSet is returned when ASDF_DOWNLOAD_PATH is missing.
	errASDFDownloadPathNotSet = errors.New("ASDF_DOWNLOAD_PATH not set")
	// errASDFInstallPathNotSet is returned when ASDF_INSTALL_PATH is missing.
	errASDFInstallPathNotSet = errors.New("ASDF_INSTALL_PATH not set")
	// errLegacyFilePathRequired is returned when no legacy file path is provided.
	errLegacyFilePathRequired = errors.New("legacy file path required")
	// errChecksumMismatch is returned when a recorded checksum does not match.
	errChecksumMismatch = errors.New("checksum mismatch")

	// version, commit and date are set via ldflags at build time by the release
	// tooling. These fields are surfaced via the "version" subcommand.
	version = "dev"
	// commit set via ldflags at build time by the release tooling.
	commit = "none" //nolint:gochecknoglobals // build metadata set via ldflags
	// date set via ldflags at build time by the release tooling.
	date = "unknown" //nolint:gochecknoglobals // build metadata set via ldflags
)

// main is the entry point for the universal-asdf-plugins.
// It initializes the CLI and executes the requested subcommand.
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run parses command-line arguments and dispatches to the appropriate handler.
func run() error {
	if len(os.Args) < 2 {
		return printUsage()
	}

	pluginName := os.Getenv("ASDF_PLUGIN_NAME")
	if pluginName == "" {
		execName := filepath.Base(os.Args[0])
		switch {
		case strings.Contains(execName, "golang"), strings.Contains(execName, "go"):
			pluginName = "golang"
		case strings.Contains(execName, "python"):
			pluginName = "python"
		case strings.Contains(execName, "nodejs"), strings.Contains(execName, "node"):
			pluginName = "nodejs"
		}
	}

	command := os.Args[1]

	switch command {
	case "version", "--version", "-v":
		fmt.Printf("universal-asdf-plugin %s (commit: %s, built: %s)\n", version, commit, date)
		return nil

	case "help", "--help", "-h":
		return printUsage()
	case "plugins":
		fmt.Println("Available plugins:")
		fmt.Println("  argo          - Argo Workflows CLI")
		fmt.Println("  argocd        - ArgoCD CLI")
		fmt.Println("  argo-rollouts - Argo Rollouts CLI")
		fmt.Println("  aws-nuke      - AWS resource cleanup")
		fmt.Println("  aws-sso-cli   - AWS SSO CLI")
		fmt.Println("  awscli        - AWS Command Line Interface")
		fmt.Println("  buf           - Protocol Buffers tooling")
		fmt.Println("  checkov       - IaC security scanner")
		fmt.Println("  cmake         - Build system generator")
		fmt.Println("  cosign        - Cosign container signing")
		fmt.Println("  doctl         - DigitalOcean CLI")
		fmt.Println("  gcloud        - Google Cloud SDK")
		fmt.Println("  jq            - Command-line JSON processor")
		fmt.Println("  k9s           - Kubernetes CLI")
		fmt.Println("  kind          - Kubernetes IN Docker")
		fmt.Println("  ko            - Container image builder")
		fmt.Println("  kubectl       - Kubernetes CLI")
		fmt.Println("  lazygit       - Git TUI")
		fmt.Println("  linkerd       - Service mesh")
		fmt.Println("  nerdctl       - Docker-compatible CLI")
		fmt.Println("  ginkgo        - BDD testing framework")
		fmt.Println("  github-cli    - GitHub CLI")
		fmt.Println("  gitsign       - Git commit signing")
		fmt.Println("  gitleaks      - Detect secrets in code")
		fmt.Println("  golang        - Go programming language")
		fmt.Println("  goreleaser    - Release automation")
		fmt.Println("  golangci-lint - Go linters aggregator")
		fmt.Println("  grype         - Vulnerability scanner")
		fmt.Println("  helm          - Kubernetes Package Manager")
		fmt.Println("  pipx          - Python app installer")
		fmt.Println("  python        - Python programming language")
		fmt.Println("  rust          - Rust programming language")
		fmt.Println("  sccache       - Compiler cache")
		fmt.Println("  shellcheck    - Shell script analysis")
		fmt.Println("  shfmt         - Shell script formatter")
		fmt.Println("  sops          - Secrets management")
		fmt.Println("  syft          - SBOM generator")
		fmt.Println("  terraform     - Infrastructure as Code")
		fmt.Println("  tflint        - Terraform linter")
		fmt.Println("  trivy         - Container vulnerability scanner")
		fmt.Println("  terragrunt    - Terraform wrapper")
		fmt.Println("  terrascan     - IaC security scanner")
		fmt.Println("  tfupdate      - Terraform updater")
		fmt.Println("  vultr-cli     - Vultr CLI")
		fmt.Println("  nodejs        - Node.js runtime")
		fmt.Println("  opentofu      - Open source Terraform")
		fmt.Println("  protoc        - Protocol Buffers compiler")
		fmt.Println("  protoc-gen-go - Go protobuf generator")
		fmt.Println("  protoc-gen-go-grpc - gRPC Go protoc plugin")
		fmt.Println("  protoc-gen-grpc-web - gRPC-Web protoc plugin")
		fmt.Println("  protolint     - Protocol Buffers linter")
		fmt.Println("  sqlc          - SQL code generator")
		fmt.Println("  tekton-cli    - Tekton CLI")
		fmt.Println("  telepresence  - K8s local dev")
		fmt.Println("  traefik       - Cloud native proxy")
		fmt.Println("  velero        - Kubernetes backup")
		fmt.Println("  upx           - Executable packer")
		fmt.Println("  uv            - Python package manager")
		fmt.Println("  yq            - YAML processor")
		fmt.Println("  zig           - Zig programming language")

		return nil

	case "install-plugin":
		return cmdInstallPlugin()

	case "update-tool-versions":
		return cmdUpdateToolVersions()

	case "generate-tool-sums":
		return cmdGenerateToolSums()
	}

	if pluginName == "" && len(os.Args) >= 3 {
		pluginName = os.Args[2]
	}

	if pluginName == "" {
		return errPluginNameRequired
	}

	plugin, err := plugins.GetPlugin(pluginName)
	if err != nil {
		return err
	}

	ctx := context.Background()

	switch command {
	case "list-all":
		return cmdListAll(ctx, plugin)
	case "download":
		return cmdDownload(ctx, plugin)
	case "install":
		return cmdInstall(ctx, plugin)
	case "uninstall":
		return cmdUninstall(ctx, plugin)
	case "list-bin-paths":
		return cmdListBinPaths(plugin)
	case "exec-env":
		return cmdExecEnv(plugin)
	case "latest-stable":
		return cmdLatestStable(ctx, plugin)
	case "list-legacy-filenames":
		return cmdListLegacyFilenames(plugin)
	case "parse-legacy-file":
		return cmdParseLegacyFile(plugin)
	case "help.overview":
		return cmdHelpOverview(plugin)
	case "help.deps":
		return cmdHelpDeps(plugin)
	case "help.config":
		return cmdHelpConfig(plugin)
	case "help.links":
		return cmdHelpLinks(plugin)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, command)
	}
}

// printUsage displays the CLI usage information to stdout.
func printUsage() error {
	fmt.Print(`universal-asdf-plugin - Universal ASDF plugin implementation in Go

Usage:
  universal-asdf-plugin <command> [plugin] [options]

Commands:
  list-all              List all available versions for a plugin
  download              Download a specific version (verifies/records checksums)
  install               Install a specific version
  list-bin-paths        List binary paths for installed version
  exec-env              Print environment variables for execution
  plugins               List available plugins
  install-plugin        Install this binary as asdf plugin(s)
  update-tool-versions  Update .tool-versions, replacing 'latest' with actual versions
  version               Print version information
  help                  Print this help message

Plugins:
  golang         Go programming language
  python         Python programming language
  nodejs         Node.js runtime

Environment Variables:
  ASDF_PLUGIN_NAME      Plugin name (golang, python, nodejs)
  ASDF_INSTALL_VERSION  Version to install
  ASDF_INSTALL_TYPE     Installation type (version or ref)
  ASDF_INSTALL_PATH     Installation path
  ASDF_DOWNLOAD_PATH    Download path

Plugin-specific Environment Variables:
  Go:
    ASDF_GOLANG_SKIP_CHECKSUM          Skip checksum verification
    ASDF_GOLANG_DEFAULT_PACKAGES_FILE  Path to default packages file
    ASDF_GOLANG_OVERWRITE_ARCH         Override architecture detection

  Python:
    ASDF_PYTHON_PATCH_URL              URL to patch file
    ASDF_PYTHON_PATCHES_DIRECTORY      Directory containing patch files
    ASDF_PYTHON_DEFAULT_PACKAGES_FILE  Path to default packages file

  Node.js:
    ASDF_NPM_DEFAULT_PACKAGES_FILE     Path to default npm packages file
    ASDF_NODEJS_AUTO_ENABLE_COREPACK   Enable corepack automatically

Examples:
  # List all Go versions
  ASDF_PLUGIN_NAME=golang universal-asdf-plugin list-all

  # Install Go 1.21.0
  ASDF_PLUGIN_NAME=golang ASDF_INSTALL_VERSION=1.21.0 \
    ASDF_INSTALL_PATH=~/.asdf/installs/golang/1.21.0 \
    universal-asdf-plugin install

  # Or specify plugin as argument
  universal-asdf-plugin list-all golang
`)

	return nil
}

// cmdListAll implements the `list-all` subcommand for a plugins.
func cmdListAll(ctx context.Context, plugin asdf.Plugin) error {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("listing versions: %w", err)
	}

	fmt.Println(strings.Join(versions, " "))

	return nil
}

// cmdDownload implements the `download` subcommand for a plugins.
// It downloads the requested version into ASDF_DOWNLOAD_PATH and manages checksums.
func cmdDownload(ctx context.Context, plugin asdf.Plugin) error {
	installVersion := os.Getenv("ASDF_INSTALL_VERSION")
	if installVersion == "" {
		return errASDFInstallVersionNotSet
	}

	downloadPath := os.Getenv("ASDF_DOWNLOAD_PATH")
	if downloadPath == "" {
		return errASDFDownloadPathNotSet
	}

	if err := os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	if err := plugin.Download(ctx, installVersion, downloadPath); err != nil {
		return err
	}

	if err := verifyToolSum(plugin.Name(), installVersion, downloadPath); err != nil {
		return err
	}

	if err := recordToolSum(plugin.Name(), installVersion, downloadPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record checksum: %v\n", err)
	}

	return nil
}

// cmdInstall implements the `install` subcommand for a plugins.
// It installs the requested version into ASDF_INSTALL_PATH.
func cmdInstall(ctx context.Context, plugin asdf.Plugin) error {
	installVersion := os.Getenv("ASDF_INSTALL_VERSION")
	if installVersion == "" {
		return errASDFInstallVersionNotSet
	}

	installPath := os.Getenv("ASDF_INSTALL_PATH")
	if installPath == "" {
		return errASDFInstallPathNotSet
	}

	downloadPath := os.Getenv("ASDF_DOWNLOAD_PATH")
	if downloadPath == "" {
		downloadPath = filepath.Join(os.TempDir(), fmt.Sprintf("asdf-%s-%s", plugin.Name(), installVersion))
	}

	if err := os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	if err := os.MkdirAll(installPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	return plugin.Install(ctx, installVersion, downloadPath, installPath)
}

// cmdListBinPaths implements the `list-bin-paths` subcommand.
// It prints the plugin's binary paths for the current installation.
func cmdListBinPaths(plugin asdf.Plugin) error {
	fmt.Println(plugin.ListBinPaths())

	return nil
}

// cmdExecEnv implements the `exec-env` subcommand.
// It prints shell export statements for the plugin's execution environment.
func cmdExecEnv(plugin asdf.Plugin) error {
	installPath := os.Getenv("ASDF_INSTALL_PATH")
	if installPath == "" {
		return nil
	}

	env := plugin.ExecEnv(installPath)
	for key, value := range env {
		fmt.Printf("export %s=%q\n", key, value)
	}

	return nil
}

// cmdUninstall implements the `uninstall` subcommand.
// It removes the plugin installation at ASDF_INSTALL_PATH.
func cmdUninstall(ctx context.Context, plugin asdf.Plugin) error {
	installPath := os.Getenv("ASDF_INSTALL_PATH")
	if installPath == "" {
		return errASDFInstallPathNotSet
	}

	return plugin.Uninstall(ctx, installPath)
}

// cmdLatestStable implements the `latest-stable` subcommand.
// It prints the latest stable version matching an optional query.
func cmdLatestStable(ctx context.Context, plugin asdf.Plugin) error {
	query := ""
	if len(os.Args) >= 3 {
		query = os.Args[2]
	}

	version, err := plugin.LatestStable(ctx, query)
	if err != nil {
		return err
	}

	fmt.Println(version)

	return nil
}

// cmdListLegacyFilenames implements the `list-legacy-filenames` subcommand.
// It prints the legacy version file names recognized by the plugins.
func cmdListLegacyFilenames(plugin asdf.Plugin) error {
	filenames := plugin.ListLegacyFilenames()
	fmt.Println(strings.Join(filenames, " "))

	return nil
}

// cmdParseLegacyFile implements the `parse-legacy-file` subcommand.
// It reads a legacy version file and prints the parsed version.
func cmdParseLegacyFile(plugin asdf.Plugin) error {
	if len(os.Args) < 3 {
		return errLegacyFilePathRequired
	}

	filePath := os.Args[2]

	version, err := plugin.ParseLegacyFile(filePath)
	if err != nil {
		return err
	}

	fmt.Println(version)

	return nil
}

// cmdHelpOverview prints the plugin's overview help section.
func cmdHelpOverview(plugin asdf.Plugin) error {
	help := plugin.Help()
	fmt.Println(help.Overview)

	return nil
}

// cmdHelpDeps prints the plugin's dependency help section.
func cmdHelpDeps(plugin asdf.Plugin) error {
	help := plugin.Help()
	fmt.Println(help.Deps)

	return nil
}

// cmdHelpConfig prints the plugin's configuration help section.
func cmdHelpConfig(plugin asdf.Plugin) error {
	help := plugin.Help()
	fmt.Println(help.Config)

	return nil
}

// cmdHelpLinks prints helpful links for the plugins.
func cmdHelpLinks(plugin asdf.Plugin) error {
	help := plugin.Help()
	fmt.Println(help.Links)

	return nil
}

// cmdInstallPlugin installs this binary as one or more asdf plugins.
func cmdInstallPlugin() error {
	pluginsToInstall := asdf.AvailablePlugins()
	if len(os.Args) >= 3 {
		pluginsToInstall = os.Args[2:]
	}

	bootstrappingAsdf := slices.Contains(pluginsToInstall, "asdf")

	if bootstrappingAsdf {
		if asdf_plugin_asdf.IsAsdfInstalled() {
			fmt.Println("asdf is already installed in", asdf_plugin_asdf.GetShimsDir())

			if !asdf_plugin_asdf.IsAsdfInPath() {
				fmt.Println("\nNote: asdf shims directory is not in your PATH.")
				fmt.Println("Add the following to your shell configuration:")
				fmt.Println(asdf_plugin_asdf.GetShellConfigInstructions(detectCurrentShell()))
			}
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	installer, err := asdf.NewPluginInstaller(execPath, "")
	if err != nil {
		return err
	}

	for _, pluginName := range pluginsToInstall {
		if _, err := plugins.GetPlugin(pluginName); err != nil {
			return err
		}

		if err := installer.Install(pluginName); err != nil {
			return err
		}

		pluginDir := filepath.Join(installer.PluginsDir, pluginName)
		fmt.Printf("Installed plugin '%s' to %s\n", pluginName, pluginDir)
	}

	return nil
}

// detectCurrentShell detects the current shell from environment.
func detectCurrentShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		base := filepath.Base(shell)
		switch base {
		case "bash", "zsh", "fish", "elvish", "nu", "pwsh":
			return base
		}
	}

	return "bash"
}

// ToolUpdateResult represents the result of updating a single tool.
type ToolUpdateResult struct {
	Error      error
	Name       string
	OldVersion string
	NewVersion string
	Changed    bool
}

// cmdUpdateToolVersions updates all tools in .tool-versions to their latest versions.
// Tools with "latest" as their version will be resolved to actual version numbers.
// cmdUpdateToolVersions implements the update-tool-versions subcommand.
// It expands any "latest" entries in .tool-versions to concrete versions
// by querying each plugin for its latest stable release.
func cmdUpdateToolVersions() error {
	toolVersionsPath := ".tool-versions"
	if len(os.Args) >= 3 {
		toolVersionsPath = os.Args[2]
	}

	existingVersions, err := parseToolVersions(toolVersionsPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", toolVersionsPath, err)
	}

	if len(existingVersions) == 0 {
		fmt.Println("No tools found in", toolVersionsPath)
		return nil
	}

	ctx := context.Background()
	results := make([]ToolUpdateResult, 0, len(existingVersions))
	updatedVersions := make(map[string]string, len(existingVersions))

	// Fetch latest versions in parallel
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	type job struct {
		name       string
		oldVersion string
	}

	jobs := make([]job, 0, len(existingVersions))
	for name, version := range existingVersions {
		jobs = append(jobs, job{name: name, oldVersion: version})
	}

	for i := range jobs {
		toolJob := jobs[i]

		wg.Go(func() {
			result := ToolUpdateResult{
				Name:       toolJob.name,
				OldVersion: toolJob.oldVersion,
			}

			plugin, err := plugins.GetPlugin(toolJob.name)
			if err != nil {
				result.NewVersion = toolJob.oldVersion
				result.Error = err

				mu.Lock()

				results = append(results, result)
				updatedVersions[toolJob.name] = toolJob.oldVersion

				mu.Unlock()

				return
			}

			if toolJob.oldVersion == "latest" {
				latestVersion, err := plugin.LatestStable(ctx, "")
				if err != nil {
					result.NewVersion = toolJob.oldVersion
					result.Error = err

					mu.Lock()

					results = append(results, result)
					updatedVersions[toolJob.name] = toolJob.oldVersion

					mu.Unlock()

					return
				}

				result.NewVersion = latestVersion
				result.Changed = true
			} else {
				result.NewVersion = toolJob.oldVersion
				result.Changed = false
			}

			mu.Lock()

			results = append(results, result)
			updatedVersions[toolJob.name] = result.NewVersion

			mu.Unlock()
		})
	}

	wg.Wait()

	err = writeToolVersions(toolVersionsPath, updatedVersions)
	if err != nil {
		return fmt.Errorf("writing %s: %w", toolVersionsPath, err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	var updated, failed, unchanged int
	for i := range results {
		res := results[i]

		switch {
		case res.Error != nil:
			fmt.Printf("  %-20s %s (error: %v)\n", res.Name, res.OldVersion, res.Error)

			failed++

		case res.Changed:
			fmt.Printf("  %-20s %s -> %s\n", res.Name, res.OldVersion, res.NewVersion)

			updated++

		default:
			unchanged++
		}
	}

	fmt.Printf("\nUpdated: %d, Unchanged: %d, Failed: %d\n", updated, unchanged, failed)

	return nil
}

// parseToolVersions parses a .tool-versions file and returns a map of tool name to version.
// parseToolVersions reads a .tool-versions file into a map keyed by tool
// name so that callers can update or inspect requested versions.
func parseToolVersions(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	versions := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			versions[fields[0]] = fields[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return versions, nil
}

// writeToolVersions writes the given versions map to a .tool-versions file.
// writeToolVersions writes the provided versions back to a .tool-versions
// file, keeping tools sorted for deterministic output.
func writeToolVersions(path string, versions map[string]string) error {
	keys := make([]string, 0, len(versions))
	for name := range versions {
		keys = append(keys, name)
	}

	sort.Strings(keys)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, asdf.CommonFilePermission)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, name := range keys {
		if toolVersion := versions[name]; toolVersion != "" {
			fmt.Fprintf(file, "%s %s\n", name, toolVersion)
		}
	}

	return nil
}

// toolSumsFile is the filename used to store checksums for helper tools.
const toolSumsFile = ".tool-sums"

// parseToolSums parses a .tool-sums file and returns a map of "name:version" to hash.
// parseToolSums loads a .tool-sums file into a map of "name:version" to
// recorded checksum so downloads can be verified later.
func parseToolSums(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}

		return nil, err
	}
	defer file.Close()

	sums := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			key := fields[0] + ":" + fields[1]

			sums[key] = fields[2]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return sums, nil
}

// writeToolSums writes the tool sums to a .tool-sums file.
func writeToolSums(path string, sums map[string]string) error {
	// Parse keys to sort by name then version
	type entry struct {
		name    string
		version string
		hash    string
	}

	entries := make([]entry, 0, len(sums))
	for key, hash := range sums {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			entries = append(entries, entry{name: parts[0], version: parts[1], hash: hash})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].name != entries[j].name {
			return entries[i].name < entries[j].name
		}

		return entries[i].version < entries[j].version
	})

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "# Tool checksums - DO NOT EDIT")
	fmt.Fprintln(file, "# Format: name version sha256:hash")

	for i := range entries {
		fmt.Fprintf(file, "%s %s %s\n", entries[i].name, entries[i].version, entries[i].hash)
	}

	return nil
}

// calculateFileHash calculates the SHA256 hash of a file.
func calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

// calculateDirHash calculates a combined hash of all files in a directory.
func calculateDirHash(dir string) (string, error) {
	hash := sha256.New()

	err := filepath.WalkDir(dir, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return nil
		}

		info, err := dirEntry.Info()
		if err != nil {
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil
			}

			hash.Write([]byte(relPath + "->" + target))

			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		hash.Write([]byte(relPath))

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		if _, err := io.Copy(hash, file); err != nil {
			return nil
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

// getDownloadHash calculates the hash of downloaded files in the download path.
func getDownloadHash(downloadPath string) (string, error) {
	entries, err := os.ReadDir(downloadPath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".tar.gz") ||
			strings.HasSuffix(name, ".tar.xz") ||
			strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".gz") {
			return calculateFileHash(filepath.Join(downloadPath, name))
		}
	}

	return calculateDirHash(downloadPath)
}

// verifyToolSum verifies the checksum of a downloaded tool.
func verifyToolSum(name, version, downloadPath string) error {
	sumsPath := toolSumsFile

	sums, err := parseToolSums(sumsPath)
	if err != nil {
		return fmt.Errorf("reading tool sums: %w", err)
	}

	key := name + ":" + version

	expectedHash, exists := sums[key]
	if !exists {
		return nil
	}

	actualHash, err := getDownloadHash(downloadPath)
	if err != nil {
		return fmt.Errorf("calculating hash: %w", err)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("%w for %s %s: expected %s, got %s", errChecksumMismatch, name, version, expectedHash, actualHash)
	}

	return nil
}

// recordToolSum records the checksum of a downloaded tool.
func recordToolSum(name, version, downloadPath string) error {
	sumsPath := toolSumsFile

	sums, err := parseToolSums(sumsPath)
	if err != nil {
		return fmt.Errorf("reading tool sums: %w", err)
	}

	hash, err := getDownloadHash(downloadPath)
	if err != nil {
		return fmt.Errorf("calculating hash: %w", err)
	}

	key := name + ":" + version

	sums[key] = hash

	if err := writeToolSums(sumsPath, sums); err != nil {
		return fmt.Errorf("writing tool sums: %w", err)
	}

	return nil
}

// cmdGenerateToolSums generates checksums for all installed tools (internal command for selftest).
func cmdGenerateToolSums() error {
	toolVersionsPath := ".tool-versions"

	versions, err := parseToolVersions(toolVersionsPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", toolVersionsPath, err)
	}

	if len(versions) == 0 {
		return nil
	}

	asdfDataDir := os.Getenv("ASDF_DATA_DIR")
	if asdfDataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}

		asdfDataDir = filepath.Join(home, ".asdf")
	}

	sums := make(map[string]string)

	for name, version := range versions {
		if version == "nightly" || version == "latest" {
			continue
		}

		installPath := filepath.Join(asdfDataDir, "installs", name, version)
		if _, err := os.Stat(installPath); os.IsNotExist(err) {
			continue
		}

		hash, err := calculateDirHash(installPath)
		if err != nil {
			continue
		}

		key := name + ":" + version

		sums[key] = hash
	}

	if err := writeToolSums(toolSumsFile, sums); err != nil {
		return fmt.Errorf("writing %s: %w", toolSumsFile, err)
	}

	fmt.Printf("Generated checksums for %d tools\n", len(sums))

	return nil
}
