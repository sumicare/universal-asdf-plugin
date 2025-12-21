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

	p "github.com/sumicare/universal-asdf-plugin/plugins"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/plugins"
	"github.com/urfave/cli/v2"
)

var (
	// errPluginNameRequired is returned when no plugin name can be determined.
	errPluginNameRequired = errors.New(
		"plugin name required. Set ASDF_PLUGIN_NAME or specify as argument",
	)
	// errASDFInstallVersionNotSet is returned when ASDF_INSTALL_VERSION is missing.
	errASDFInstallVersionNotSet = errors.New("ASDF_INSTALL_VERSION not set")
	// errASDFDownloadPathNotSet is returned when ASDF_DOWNLOAD_PATH is missing.
	errASDFDownloadPathNotSet = errors.New("ASDF_DOWNLOAD_PATH not set")
	// errASDFInstallPathNotSet is returned when ASDF_INSTALL_PATH is missing.
	errASDFInstallPathNotSet = errors.New("ASDF_INSTALL_PATH not set")
	// errLegacyFilePathRequired is returned when no legacy file path is provided.
	errLegacyFilePathRequired = errors.New("legacy file path required")
	// errAsdfPluginCastFailed is returned when casting to AsdfPlugin fails.
	errAsdfPluginCastFailed = errors.New("failed to cast to AsdfPlugin")
	// errChecksumMismatch is returned when a recorded checksum does not match.
	errChecksumMismatch = errors.New("checksum mismatch")
	// errWhichUsage indicates invalid usage of the which command.
	errWhichUsage = errors.New("usage: asdf which <tool>")
	// errNoVersionSet is returned when no version is configured for a tool.
	errNoVersionSet = errors.New("no version set")
	// errVersionNotInstalled is returned when a version is not installed.
	errVersionNotInstalled = errors.New("version is not installed")
	// errNoExecutableFound is returned when no executable can be located in an install.
	errNoExecutableFound = errors.New("no executable found")

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
	app := newCLIApp()

	args := reorderFlags(os.Args)

	err := app.Run(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// reorderFlags moves command-level flags to appear before positional arguments.
// This works around urfave/cli's requirement that flags come before args.
// Keeps the command name in place to avoid triggering global flags.
func reorderFlags(args []string) []string {
	if len(args) < 3 {
		return args
	}

	result := make([]string, 0, len(args))

	result = append(result, args[0])

	var cmdIdx int

	for i := 1; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			cmdIdx = i

			break
		}
	}

	if cmdIdx == 0 {
		return args
	}

	result = append(result, args[1:cmdIdx]...)

	if cmdIdx >= len(args) {
		return result
	}

	result = append(result, args[cmdIdx])
	cmdIdx++

	var (
		flags       []string
		positionals []string
	)

	for i := cmdIdx; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") &&
				!strings.Contains(args[i], "=") {
				i++

				flags = append(flags, args[i])
			}
		} else {
			positionals = append(positionals, args[i])
		}
	}

	result = append(result, flags...)
	result = append(result, positionals...)

	return result
}

// newCLIApp builds the urfave/cli application.
func newCLIApp() *cli.App {
	pluginFlag := &cli.StringFlag{
		Name:    "plugin",
		Aliases: []string{"p"},
		Usage:   "plugin name (e.g., golang, python, nodejs)",
		EnvVars: []string{"ASDF_PLUGIN_NAME"},
	}

	versionFlag := &cli.StringFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "version to install/download",
		EnvVars: []string{"ASDF_INSTALL_VERSION"},
	}

	downloadPathFlag := &cli.StringFlag{
		Name:    "download-path",
		Usage:   "path to store downloads",
		EnvVars: []string{"ASDF_DOWNLOAD_PATH"},
	}

	installPathFlag := &cli.StringFlag{
		Name:    "install-path",
		Usage:   "installation path",
		EnvVars: []string{"ASDF_INSTALL_PATH"},
	}

	queryFlag := &cli.StringFlag{
		Name:    "query",
		Aliases: []string{"q"},
		Usage:   "filter for latest-stable (optional)",
	}

	legacyFileFlag := &cli.StringFlag{
		Name:    "file",
		Aliases: []string{"f"},
		Usage:   "path to legacy version file",
	}

	return &cli.App{
		Name:    "universal-asdf-plugin",
		Usage:   "universal ASDF plugin implementation in Go",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Flags: []cli.Flag{
			pluginFlag,
		},
		Commands: []*cli.Command{
			{
				Name:  "plugins",
				Usage: "List available plugins",
				Action: func(_ *cli.Context) error {
					_, _ = fmt.Fprintln(os.Stdout, "Available plugins:")
					_, _ = fmt.Fprintln(os.Stdout, "  argo          - Argo Workflows CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  argocd        - ArgoCD CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  argo-rollouts - Argo Rollouts CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  aws-nuke      - AWS resource cleanup")
					_, _ = fmt.Fprintln(os.Stdout, "  aws-sso-cli   - AWS SSO CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  awscli        - AWS Command Line Interface")
					_, _ = fmt.Fprintln(os.Stdout, "  buf           - Protocol Buffers tooling")
					_, _ = fmt.Fprintln(os.Stdout, "  checkov       - IaC security scanner")
					_, _ = fmt.Fprintln(os.Stdout, "  cmake         - Build system generator")
					_, _ = fmt.Fprintln(os.Stdout, "  cosign        - Cosign container signing")
					_, _ = fmt.Fprintln(os.Stdout, "  doctl         - DigitalOcean CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  gcloud        - Google Cloud SDK")
					_, _ = fmt.Fprintln(os.Stdout, "  jq            - Command-line JSON processor")
					_, _ = fmt.Fprintln(os.Stdout, "  k9s           - Kubernetes CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  kind          - Kubernetes IN Docker")
					_, _ = fmt.Fprintln(os.Stdout, "  ko            - Container image builder")
					_, _ = fmt.Fprintln(os.Stdout, "  kubectl       - Kubernetes CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  lazygit       - Git TUI")
					_, _ = fmt.Fprintln(os.Stdout, "  linkerd       - Service mesh")
					_, _ = fmt.Fprintln(os.Stdout, "  nerdctl       - Docker-compatible CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  ginkgo        - BDD testing framework")
					_, _ = fmt.Fprintln(os.Stdout, "  github-cli    - GitHub CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  gitsign       - Git commit signing")
					_, _ = fmt.Fprintln(os.Stdout, "  gitleaks      - Detect secrets in code")
					_, _ = fmt.Fprintln(os.Stdout, "  golang        - Go programming language")
					_, _ = fmt.Fprintln(os.Stdout, "  goreleaser    - Release automation")
					_, _ = fmt.Fprintln(os.Stdout, "  golangci-lint - Go linters aggregator")
					_, _ = fmt.Fprintln(os.Stdout, "  grype         - Vulnerability scanner")
					_, _ = fmt.Fprintln(os.Stdout, "  helm          - Kubernetes Package Manager")
					_, _ = fmt.Fprintln(os.Stdout, "  pipx          - Python app installer")
					_, _ = fmt.Fprintln(os.Stdout, "  python        - Python programming language")
					_, _ = fmt.Fprintln(os.Stdout, "  rust          - Rust programming language")
					_, _ = fmt.Fprintln(os.Stdout, "  sccache       - Compiler cache")
					_, _ = fmt.Fprintln(os.Stdout, "  shellcheck    - Shell script analysis")
					_, _ = fmt.Fprintln(os.Stdout, "  shfmt         - Shell script formatter")
					_, _ = fmt.Fprintln(os.Stdout, "  sops          - Secrets management")
					_, _ = fmt.Fprintln(os.Stdout, "  syft          - SBOM generator")
					_, _ = fmt.Fprintln(os.Stdout, "  terraform     - Infrastructure as Code")
					_, _ = fmt.Fprintln(os.Stdout, "  tflint        - Terraform linter")
					_, _ = fmt.Fprintln(
						os.Stdout,
						"  trivy         - Container vulnerability scanner",
					)
					_, _ = fmt.Fprintln(os.Stdout, "  terragrunt    - Terraform wrapper")
					_, _ = fmt.Fprintln(os.Stdout, "  terrascan     - IaC security scanner")
					_, _ = fmt.Fprintln(os.Stdout, "  tfupdate      - Terraform updater")
					_, _ = fmt.Fprintln(os.Stdout, "  vultr-cli     - Vultr CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  nodejs        - Node.js runtime")
					_, _ = fmt.Fprintln(os.Stdout, "  opentofu      - Open source Terraform")
					_, _ = fmt.Fprintln(os.Stdout, "  protoc        - Protocol Buffers compiler")
					_, _ = fmt.Fprintln(os.Stdout, "  protoc-gen-go - Go protobuf generator")
					_, _ = fmt.Fprintln(os.Stdout, "  protoc-gen-go-grpc - gRPC Go protoc plugin")
					_, _ = fmt.Fprintln(os.Stdout, "  protoc-gen-grpc-web - gRPC-Web protoc plugin")
					_, _ = fmt.Fprintln(os.Stdout, "  protolint     - Protocol Buffers linter")
					_, _ = fmt.Fprintln(os.Stdout, "  sqlc          - SQL code generator")
					_, _ = fmt.Fprintln(os.Stdout, "  tekton-cli    - Tekton CLI")
					_, _ = fmt.Fprintln(os.Stdout, "  telepresence  - K8s local dev")
					_, _ = fmt.Fprintln(os.Stdout, "  traefik       - Cloud native proxy")
					_, _ = fmt.Fprintln(os.Stdout, "  velero        - Kubernetes backup")
					_, _ = fmt.Fprintln(os.Stdout, "  upx           - Executable packer")
					_, _ = fmt.Fprintln(os.Stdout, "  uv            - Python package manager")
					_, _ = fmt.Fprintln(os.Stdout, "  yq            - YAML processor")
					_, _ = fmt.Fprintln(os.Stdout, "  zig           - Zig programming language")

					return nil
				},
			},
			{
				Name:  "install-plugin",
				Usage: "Install this binary as asdf plugin(s)",
				Action: func(_ *cli.Context) error {
					return cmdInstallPlugin()
				},
			},
			{
				Name:  "update-tool-versions",
				Usage: "Update .tool-versions, replacing 'latest' with actual versions",
				Action: func(_ *cli.Context) error {
					return cmdUpdateToolVersions()
				},
			},
			{
				Name:  "generate-tool-sums",
				Usage: "Generate tool checksum records",
				Action: func(_ *cli.Context) error {
					return cmdGenerateToolSums()
				},
			},
			{
				Name:  "reshim",
				Usage: "Regenerate shims for all installed tool versions",
				Action: func(_ *cli.Context) error {
					return cmdReshim()
				},
			},
			{
				Name:  "which",
				Usage: "Display the path to an executable",
				Flags: []cli.Flag{pluginFlag, versionFlag},
				Action: func(cliContext *cli.Context) error {
					toolName := cliContext.Args().First()
					if toolName == "" {
						// Try to resolve from flags/context
						plugin, _, err := resolvePluginFromContext(cliContext)
						if err == nil {
							toolName = plugin.Name()
						}
					}

					if toolName == "" {
						return errWhichUsage
					}

					return cmdWhich(toolName)
				},
			},
			{
				Name:  "list-all",
				Usage: "List all available versions for a plugin",
				Flags: []cli.Flag{pluginFlag},
				Action: func(c *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(c)
					if err != nil {
						return err
					}

					return cmdListAll(c.Context, plugin)
				},
			},
			{
				Name:  "download",
				Usage: "Download a specific version (verifies/records checksums)",
				Flags: []cli.Flag{pluginFlag, versionFlag, downloadPathFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, args, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					installVersion := cliContext.String("version")
					if installVersion == "" && len(args) > 0 {
						installVersion = args[0]
					}

					if installVersion == "" {
						latestVersion, err := plugin.LatestStable(cliContext.Context, "")
						if err != nil {
							return fmt.Errorf("resolving latest version: %w", err)
						}

						installVersion = latestVersion
					}

					downloadPath := cliContext.String("download-path")
					if downloadPath == "" {
						downloadPath = filepath.Join(
							getAsdfDataDir(),
							"downloads",
							plugin.Name(),
							installVersion,
						)
					}

					return cmdDownload(cliContext.Context, plugin, installVersion, downloadPath)
				},
			},
			{
				Name:  "install",
				Usage: "Install a specific version",
				Flags: []cli.Flag{pluginFlag, versionFlag, downloadPathFlag, installPathFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, args, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					installVersion := cliContext.String("version")
					if installVersion == "" && len(args) > 0 {
						installVersion = args[0]
					}

					if installVersion == "" {
						latestVersion, err := plugin.LatestStable(cliContext.Context, "")
						if err != nil {
							return fmt.Errorf("resolving latest version: %w", err)
						}

						installVersion = latestVersion
					}

					installPath := cliContext.String("install-path")
					if installPath == "" {
						installPath = filepath.Join(
							getAsdfDataDir(),
							"installs",
							plugin.Name(),
							installVersion,
						)
					}

					downloadPath := cliContext.String("download-path")
					if downloadPath == "" {
						downloadPath = filepath.Join(
							getAsdfDataDir(),
							"downloads",
							plugin.Name(),
							installVersion,
						)
					}

					return cmdInstall(
						cliContext.Context,
						plugin,
						installVersion,
						downloadPath,
						installPath,
					)
				},
			},
			{
				Name:  "uninstall",
				Usage: "Uninstall a specific version",
				Flags: []cli.Flag{pluginFlag, installPathFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, args, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					installPath := cliContext.String("install-path")
					if installPath == "" && len(args) > 0 {
						uninstallVersion := args[0]

						installPath = filepath.Join(
							getAsdfDataDir(),
							"installs",
							plugin.Name(),
							uninstallVersion,
						)
					}

					if installPath == "" {
						return errASDFInstallPathNotSet
					}

					return cmdUninstall(cliContext.Context, plugin, installPath)
				},
			},
			{
				Name:  "list-bin-paths",
				Usage: "List binary paths for installed version",
				Flags: []cli.Flag{pluginFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdListBinPaths(plugin)
				},
			},
			{
				Name:  "exec-env",
				Usage: "Print environment variables for execution",
				Flags: []cli.Flag{pluginFlag, installPathFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdExecEnv(plugin, cliContext.String("install-path"))
				},
			},
			{
				Name:  "latest-stable",
				Usage: "Return latest stable version",
				Flags: []cli.Flag{pluginFlag, queryFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, args, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					query := cliContext.String("query")
					if query == "" && len(args) > 0 {
						query = args[0]
					}

					return cmdLatestStable(cliContext.Context, plugin, query)
				},
			},
			{
				Name:  "list-legacy-filenames",
				Usage: "List legacy version file names",
				Flags: []cli.Flag{pluginFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdListLegacyFilenames(plugin)
				},
			},
			{
				Name:  "parse-legacy-file",
				Usage: "Parse a legacy version file",
				Flags: []cli.Flag{pluginFlag, legacyFileFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, args, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					filePath := cliContext.String("file")
					if filePath == "" && len(args) > 0 {
						filePath = args[0]
					}

					if filePath == "" {
						return errLegacyFilePathRequired
					}

					return cmdParseLegacyFile(plugin, filePath)
				},
			},
			{
				Name:  "help.overview",
				Usage: "Show plugin overview help section",
				Flags: []cli.Flag{pluginFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdHelpOverview(plugin)
				},
			},
			{
				Name:  "help.deps",
				Usage: "Show plugin dependencies help section",
				Flags: []cli.Flag{pluginFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdHelpDeps(plugin)
				},
			},
			{
				Name:  "help.config",
				Usage: "Show plugin config help section",
				Flags: []cli.Flag{pluginFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdHelpConfig(plugin)
				},
			},
			{
				Name:  "help.links",
				Usage: "Show plugin links help section",
				Flags: []cli.Flag{pluginFlag},
				Action: func(cliContext *cli.Context) error {
					plugin, _, err := resolvePluginFromContext(cliContext)
					if err != nil {
						return err
					}

					return cmdHelpLinks(plugin)
				},
			},
		},
	}
}

// getAsdfDataDir returns the ASDF data directory, defaulting to ~/.asdf if not set.
func getAsdfDataDir() string {
	if dir := os.Getenv("ASDF_DATA_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".asdf")
	}

	return filepath.Join(home, ".asdf")
}

// resolvePluginFromContext resolves plugin from flag, first arg, or executable name.
func resolvePluginFromContext(cliContext *cli.Context) (asdf.Plugin, []string, error) {
	pluginName := strings.TrimSpace(cliContext.String("plugin"))

	if pluginName == "" && cliContext.Args().Present() {
		pluginName = cliContext.Args().First()
	}

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

	if pluginName == "" {
		return nil, nil, errPluginNameRequired
	}

	plugin, err := plugins.GetPlugin(pluginName)
	if err != nil {
		return nil, nil, err
	}

	args := cliContext.Args().Slice()
	if len(args) > 0 && args[0] == pluginName {
		args = args[1:]
	}

	return plugin, args, nil
}

// cmdListAll implements the `list-all` subcommand for a plugins.
func cmdListAll(ctx context.Context, plugin asdf.Plugin) error {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("listing versions: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, strings.Join(versions, " "))

	return nil
}

// cmdDownload implements the `download` subcommand for a plugins.
// It downloads the requested version into the provided downloadPath and manages checksums.
func cmdDownload(
	ctx context.Context,
	plugin asdf.Plugin,
	installVersion, downloadPath string,
) error {
	if installVersion == "" {
		return errASDFInstallVersionNotSet
	}

	if downloadPath == "" {
		return errASDFDownloadPathNotSet
	}

	err := os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)
	if err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	err = plugin.Download(ctx, installVersion, downloadPath)
	if err != nil {
		return err
	}

	err = verifyToolSum(plugin.Name(), installVersion, downloadPath)
	if err != nil {
		return err
	}

	err = recordToolSum(plugin.Name(), installVersion, downloadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record checksum: %v\n", err)
	}

	return nil
}

// cmdInstall implements the `install` subcommand for a plugins.
// It installs the requested version into installPath.
func cmdInstall(
	ctx context.Context,
	plugin asdf.Plugin,
	installVersion, downloadPath, installPath string,
) error {
	if installVersion == "" {
		return errASDFInstallVersionNotSet
	}

	if installPath == "" {
		return errASDFInstallPathNotSet
	}

	actualDownloadPath := downloadPath
	if actualDownloadPath == "" {
		actualDownloadPath = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("asdf-%s-%s", plugin.Name(), installVersion),
		)
	}

	err := os.MkdirAll(actualDownloadPath, asdf.CommonDirectoryPermission)
	if err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	err = os.MkdirAll(installPath, asdf.CommonDirectoryPermission)
	if err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	return plugin.Install(ctx, installVersion, actualDownloadPath, installPath)
}

// cmdListBinPaths implements the `list-bin-paths` subcommand.
// It prints the plugin's binary paths for the current installation.
func cmdListBinPaths(plugin asdf.Plugin) error {
	_, _ = fmt.Fprintln(os.Stdout, plugin.ListBinPaths())

	return nil
}

// cmdExecEnv implements the `exec-env` subcommand.
// It prints shell export statements for the plugin's execution environment.
func cmdExecEnv(plugin asdf.Plugin, installPath string) error {
	if installPath == "" {
		return nil
	}

	env := plugin.ExecEnv(installPath)
	for key, value := range env {
		_, _ = fmt.Fprintf(os.Stdout, "export %s=%q\n", key, value)
	}

	return nil
}

// cmdUninstall implements the `uninstall` subcommand.
// It removes the plugin installation at ASDF_INSTALL_PATH.
func cmdUninstall(ctx context.Context, plugin asdf.Plugin, installPath string) error {
	if installPath == "" {
		return errASDFInstallPathNotSet
	}

	return plugin.Uninstall(ctx, installPath)
}

// cmdLatestStable implements the `latest-stable` subcommand.
// It prints the latest stable version matching an optional query.
func cmdLatestStable(ctx context.Context, plugin asdf.Plugin, query string) error {
	latestVersion, err := plugin.LatestStable(ctx, query)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, latestVersion)

	return nil
}

// cmdListLegacyFilenames implements the `list-legacy-filenames` subcommand.
// It prints the legacy version file names recognized by the plugins.
func cmdListLegacyFilenames(plugin asdf.Plugin) error {
	filenames := plugin.ListLegacyFilenames()
	_, _ = fmt.Fprintln(os.Stdout, strings.Join(filenames, " "))

	return nil
}

// cmdParseLegacyFile implements the `parse-legacy-file` subcommand.
// It reads a legacy version file and prints the parsed version.
func cmdParseLegacyFile(plugin asdf.Plugin, filePath string) error {
	if filePath == "" {
		return errLegacyFilePathRequired
	}

	parsedVersion, err := plugin.ParseLegacyFile(filePath)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, parsedVersion)

	return nil
}

// cmdHelpOverview prints the plugin's overview help section.
func cmdHelpOverview(plugin asdf.Plugin) error {
	help := plugin.Help()
	_, _ = fmt.Fprintln(os.Stdout, help.Overview)

	return nil
}

// cmdHelpDeps prints the plugin's dependency help section.
func cmdHelpDeps(plugin asdf.Plugin) error {
	help := plugin.Help()
	_, _ = fmt.Fprintln(os.Stdout, help.Deps)

	return nil
}

// cmdHelpConfig prints the plugin's configuration help section.
func cmdHelpConfig(plugin asdf.Plugin) error {
	help := plugin.Help()
	_, _ = fmt.Fprintln(os.Stdout, help.Config)

	return nil
}

// cmdHelpLinks prints helpful links for the plugins.
func cmdHelpLinks(plugin asdf.Plugin) error {
	help := plugin.Help()
	_, _ = fmt.Fprintln(os.Stdout, help.Links)

	return nil
}

// cmdWhich displays the path to an executable.
func cmdWhich(toolName string) error {
	ctx := context.Background()

	// 1. Resolve version
	toolVersion := resolveToolVersion(ctx, toolName)
	if toolVersion == "" {
		return fmt.Errorf("%w for %s", errNoVersionSet, toolName)
	}

	// 2. Get plugin
	plugin, err := plugins.GetPlugin(toolName)
	if err != nil {
		return err
	}

	// 3. Construct install path
	asdfDataDir := getAsdfDataDir()
	installPath := filepath.Join(asdfDataDir, "installs", toolName, toolVersion)

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s %s", errVersionNotInstalled, toolName, toolVersion)
	}

	// 4. Find executable
	binPathsStr := plugin.ListBinPaths()
	if binPathsStr == "" {
		binPathsStr = "bin"
	}

	binPaths := strings.FieldsSeq(binPathsStr)
	for binPath := range binPaths {
		binDir := filepath.Join(installPath, binPath)

		entries, err := os.ReadDir(binDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			// return the first executable found
			// simplified logic, might need to match tool name or config
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.Mode()&0o111 != 0 {
				_, _ = fmt.Fprintln(os.Stdout, filepath.Join(binDir, entry.Name()))

				return nil
			}
		}
	}

	return fmt.Errorf("%w for %s %s", errNoExecutableFound, toolName, toolVersion)
}

// cmdReshim regenerates shims for all installed tool versions.
func cmdReshim() error {
	asdfDataDir := os.Getenv("ASDF_DATA_DIR")
	if asdfDataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}

		asdfDataDir = filepath.Join(homeDir, ".asdf")
	}

	shimsDir := filepath.Join(asdfDataDir, "shims")
	installsDir := filepath.Join(asdfDataDir, "installs")

	// Ensure shims directory exists
	if err := os.MkdirAll(shimsDir, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating shims directory: %w", err)
	}

	// Remove all existing shims
	entries, err := os.ReadDir(shimsDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading shims directory: %w", err)
	}

	for _, entry := range entries {
		err := os.Remove(filepath.Join(shimsDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove old shim %s: %v\n", entry.Name(), err)
		}
	}

	// Read .tool-versions to determine which versions to shim
	toolVersions, err := parseToolVersions(".tool-versions")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .tool-versions: %w", err)
	}

	shimCount := 0

	for toolName, version := range toolVersions {
		installPath := filepath.Join(installsDir, toolName, version)

		// Skip if not installed
		if _, err := os.Stat(installPath); os.IsNotExist(err) {
			continue
		}

		// Get bin paths for this tool
		plugin, err := plugins.GetPlugin(toolName)
		if err != nil {
			continue
		}

		binPathsStr := plugin.ListBinPaths()
		if binPathsStr == "" {
			binPathsStr = "bin"
		}

		binPaths := strings.FieldsSeq(binPathsStr)
		for binPath := range binPaths {
			binDir := filepath.Join(installPath, binPath)

			binaries, err := os.ReadDir(binDir)
			if err != nil {
				continue
			}

			for _, binary := range binaries {
				if binary.IsDir() {
					continue
				}

				binFile := filepath.Join(binDir, binary.Name())

				info, err := os.Stat(binFile)
				if err != nil {
					continue
				}

				// Only create shims for executable files
				if info.Mode()&0o111 == 0 {
					continue
				}

				shimPath := filepath.Join(shimsDir, binary.Name())

				// Remove existing shim if present
				if err := os.Remove(shimPath); err != nil && !os.IsNotExist(err) {
					fmt.Fprintf(
						os.Stderr,
						"Warning: failed to remove existing shim %s: %v\n",
						shimPath,
						err,
					)
				}

				// Create symlink to actual binary
				if err := os.Symlink(binFile, shimPath); err != nil {
					fmt.Fprintf(
						os.Stderr,
						"Warning: failed to create shim for %s: %v\n",
						binary.Name(),
						err,
					)

					continue
				}

				shimCount++
			}
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Created %d shims in %s\n", shimCount, shimsDir)

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
		asdfPlugin, ok := p.NewAsdfPlugin().(*p.AsdfPlugin)
		if !ok {
			return errAsdfPluginCastFailed
		}

		if asdfPlugin.IsAsdfInstalled() {
			_, _ = fmt.Fprintln(os.Stdout, "asdf is already installed in", asdfPlugin.GetShimsDir())

			if !asdfPlugin.IsAsdfInPath() {
				_, _ = fmt.Fprintln(os.Stdout, "\nNote: asdf shims directory is not in your PATH.")
				_, _ = fmt.Fprintln(os.Stdout, "Add the following to your shell configuration:")
				_, _ = fmt.Fprintln(
					os.Stdout,
					asdfPlugin.GetShellConfigInstructions(detectCurrentShell()),
				)
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

		err := installer.Install(pluginName)
		if err != nil {
			return err
		}

		pluginDir := filepath.Join(installer.PluginsDir, pluginName)
		_, _ = fmt.Fprintf(os.Stdout, "Installed plugin '%s' to %s\n", pluginName, pluginDir)
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
		_, _ = fmt.Fprintln(os.Stdout, "No tools found in", toolVersionsPath)

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
			_, _ = fmt.Fprintf(
				os.Stdout,
				"  %-20s %s (error: %v)\n",
				res.Name,
				res.OldVersion,
				res.Error,
			)

			failed++

		case res.Changed:
			_, _ = fmt.Fprintf(
				os.Stdout,
				"  %-20s %s -> %s\n",
				res.Name,
				res.OldVersion,
				res.NewVersion,
			)

			updated++

		default:
			unchanged++
		}
	}

	_, _ = fmt.Fprintf(
		os.Stdout,
		"\nUpdated: %d, Unchanged: %d, Failed: %d\n",
		updated,
		unchanged,
		failed,
	)

	return nil
}

// resolveToolVersion resolves the version of a tool from the nearest .tool-versions file.
func resolveToolVersion(_ context.Context, toolName string) string {
	// 1. Try to find .tool-versions
	path, err := asdf.ResolveToolVersionsPath()
	if err != nil {
		return ""
	}

	// 2. Parse it
	versions, err := parseToolVersions(path)
	if err != nil {
		return ""
	}

	return versions[toolName]
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

// parseToolSumsFromReader parses tool sums from an io.Reader.
func parseToolSumsFromReader(r io.Reader) (map[string]string, error) {
	sums := make(map[string]string)

	scanner := bufio.NewScanner(r)
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

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	return sums, nil
}

// writeToolSums writes the tool sums to a .tool-sums file.
func writeToolSums(path string, sums map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return writeToolSumsToWriter(file, sums)
}

// writeToolSumsToWriter writes the tool sums to an io.Writer.
func writeToolSumsToWriter(w io.Writer, sums map[string]string) error {
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

	if _, err := fmt.Fprintln(w, "# Tool checksums - DO NOT EDIT"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# Format: name version sha256:hash"); err != nil {
		return err
	}

	for i := range entries {
		if _, err := fmt.Fprintf(w, "%s %s %s\n", entries[i].name, entries[i].version, entries[i].hash); err != nil {
			return err
		}
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

// withToolSumsLock executes the given function with a file lock held on the tool sums file.
// If readOnly is true, it acquires a shared lock. Otherwise, it acquires an exclusive lock.
func withToolSumsLock(
	path string,
	flags int,
	lockExclusive bool,
	allowMissing bool,
	fn func(file *os.File) error,
) error {
	file, err := os.OpenFile(path, flags, asdf.CommonFilePermission)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return fn(nil)
		}

		return err
	}
	defer file.Close()

	if err := lockToolSumsFile(int(file.Fd()), lockExclusive); err != nil {
		return fmt.Errorf("locking file: %w", err)
	}

	defer func() {
		unlockErr := unlockToolSumsFile(int(file.Fd()))
		if unlockErr != nil {
			fmt.Fprintf(os.Stderr, "warning: unlocking tool sums file: %v\n", unlockErr)
		}
	}()

	return fn(file)
}

func withToolSumsReadLock(path string, fn func(file *os.File) error) error {
	return withToolSumsLock(path, os.O_RDONLY, false, true, fn)
}

func withToolSumsWriteLock(path string, fn func(file *os.File) error) error {
	return withToolSumsLock(path, os.O_RDWR|os.O_CREATE, true, false, fn)
}

// verifyToolSum verifies the checksum of a downloaded tool.
func verifyToolSum(name, version, downloadPath string) error {
	sumsPath := toolSumsFile

	var sums map[string]string

	err := withToolSumsReadLock(sumsPath, func(file *os.File) error {
		if file == nil {
			sums = make(map[string]string)

			return nil
		}

		var err error

		sums, err = parseToolSumsFromReader(file)

		return err
	})
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
		return fmt.Errorf(
			"%w for %s %s: expected %s, got %s",
			errChecksumMismatch,
			name,
			version,
			expectedHash,
			actualHash,
		)
	}

	return nil
}

// recordToolSum records the checksum of a downloaded tool.
func recordToolSum(name, version, downloadPath string) error {
	sumsPath := toolSumsFile

	hash, err := getDownloadHash(downloadPath)
	if err != nil {
		return fmt.Errorf("calculating hash: %w", err)
	}

	return withToolSumsWriteLock(sumsPath, func(file *os.File) error {
		sums, err := parseToolSumsFromReader(file)
		if err != nil {
			return fmt.Errorf("reading tool sums: %w", err)
		}

		key := name + ":" + version

		sums[key] = hash

		if err := file.Truncate(0); err != nil {
			return fmt.Errorf("truncating file: %w", err)
		}

		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("seeking file: %w", err)
		}

		if err := writeToolSumsToWriter(file, sums); err != nil {
			return fmt.Errorf("writing tool sums: %w", err)
		}

		return nil
	})
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

	_, _ = fmt.Fprintf(os.Stdout, "Generated checksums for %d tools\n", len(sums))

	return nil
}
