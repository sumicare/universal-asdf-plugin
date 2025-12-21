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

package asdf

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	// osMkdirAll is os.MkdirAll for mocking.
	osMkdirAll = os.MkdirAll //nolint:gochecknoglobals // used for mocking
	// osRemoveAll is os.RemoveAll for mocking.
	osRemoveAll = os.RemoveAll //nolint:gochecknoglobals // used for mocking
	// osReadDir is os.ReadDir for mocking.
	osReadDir = os.ReadDir //nolint:gochecknoglobals // used for mocking
	// osWriteFile is os.WriteFile for mocking.
	osWriteFile = os.WriteFile //nolint:gochecknoglobals // used for mocking
	// osStat is os.Stat for mocking.
	osStat = os.Stat //nolint:gochecknoglobals // used for mocking
)

// PluginInstaller handles installing the universal-asdf-plugin as asdf plugins.
type PluginInstaller struct {
	// ExecPath is the absolute path to the universal-asdf-plugin binary.
	ExecPath string
	// PluginsDir is the asdf plugins directory.
	PluginsDir string
}

// NewPluginInstaller creates a new PluginInstaller with the given executable path.
// If pluginsDir is empty, it will be determined from ASDF_DATA_DIR or ~/.asdf/plugins.
func NewPluginInstaller(execPath, pluginsDir string) (*PluginInstaller, error) {
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("resolving executable path: %w", err)
	}

	pluginsDirToUse := pluginsDir
	if pluginsDir == "" {
		pluginsDirToUse = GetPluginsDir()
	}

	return &PluginInstaller{
		ExecPath:   resolved,
		PluginsDir: pluginsDirToUse,
	}, nil
}

// GetPluginsDir returns the asdf plugins directory.
// It checks ASDF_DATA_DIR first, then falls back to ~/.asdf/plugins.
func GetPluginsDir() string {
	if dataDir := os.Getenv("ASDF_DATA_DIR"); dataDir != "" {
		return filepath.Join(dataDir, "plugins")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".asdf", "plugins")
	}

	return filepath.Join(home, ".asdf", "plugins")
}

// Install installs the specified plugin by creating wrapper scripts in the bin directory.
func (pi *PluginInstaller) Install(pluginName string) error {
	pluginDir := filepath.Join(pi.PluginsDir, pluginName)
	binDir := filepath.Join(pluginDir, "bin")

	err := osMkdirAll(binDir, CommonDirectoryPermission)
	if err != nil {
		return fmt.Errorf("creating plugin bin directory: %w", err)
	}

	scripts := []struct {
		name    string
		command string
	}{
		{"list-all", "list-all"},
		{"download", "download"},
		{"install", "install"},
		{"uninstall", "uninstall"},
		{"list-bin-paths", "list-bin-paths"},
		{"exec-env", "exec-env"},
		{"latest-stable", "latest-stable"},
		{"list-legacy-filenames", "list-legacy-filenames"},
		{"parse-legacy-file", "parse-legacy-file"},
		{"help.overview", "help.overview"},
		{"help.deps", "help.deps"},
		{"help.config", "help.config"},
		{"help.links", "help.links"},
	}

	for i := range scripts {
		scriptPath := filepath.Join(binDir, scripts[i].name)
		scriptContent := pi.generateWrapperScript(pluginName, scripts[i].command)

		err := osWriteFile(scriptPath, []byte(scriptContent), CommonDirectoryPermission)
		if err != nil {
			return fmt.Errorf("writing script %s: %w", scripts[i].name, err)
		}
	}

	return nil
}

// InstallAll installs all available plugins.
func (pi *PluginInstaller) InstallAll() ([]string, error) {
	plugins := []string{"golang", "python", "nodejs"}
	installed := make([]string, 0, len(plugins))

	for _, pluginName := range plugins {
		err := pi.Install(pluginName)
		if err != nil {
			return installed, fmt.Errorf("installing %s: %w", pluginName, err)
		}

		installed = append(installed, pluginName)
	}

	return installed, nil
}

// Uninstall removes the specified plugin directory.
func (pi *PluginInstaller) Uninstall(pluginName string) error {
	pluginDir := filepath.Join(pi.PluginsDir, pluginName)

	err := osRemoveAll(pluginDir)
	if err != nil {
		return fmt.Errorf("removing plugin directory: %w", err)
	}

	return nil
}

// IsInstalled checks if a plugin is already installed.
func (pi *PluginInstaller) IsInstalled(pluginName string) bool {
	pluginDir := filepath.Join(pi.PluginsDir, pluginName)
	binDir := filepath.Join(pluginDir, "bin")

	info, err := osStat(binDir)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// GetInstalledPlugins returns a list of installed plugin names.
func (pi *PluginInstaller) GetInstalledPlugins() ([]string, error) {
	entries, err := osReadDir(pi.PluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading plugins directory: %w", err)
	}

	plugins := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		binDir := filepath.Join(pi.PluginsDir, entry.Name(), "bin")

		info, err := osStat(binDir)
		if err != nil || !info.IsDir() {
			continue
		}

		plugins = append(plugins, entry.Name())
	}

	return plugins, nil
}

// generateWrapperScript returns a bash wrapper script for invoking the CLI with
// ASDF_PLUGIN_NAME set for the given plugin and command.
func (pi *PluginInstaller) generateWrapperScript(pluginName, command string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

export ASDF_PLUGIN_NAME="%s"
exec "%s" "%s" "$@"
`, pluginName, pi.ExecPath, command)
}

// AvailablePlugins returns the list of plugins that can be installed.
func AvailablePlugins() []string {
	return []string{
		"argo",
		"argocd",
		"argo-rollouts",
		"aws-nuke",
		"aws-sso-cli",
		"awscli",
		"buf",
		"checkov",
		"cmake",
		"cosign",
		"doctl",
		"gcloud",
		"ginkgo",
		"github-cli",
		"gitleaks",
		"gitsign",
		"golang",
		"golangci-lint",
		"goreleaser",
		"grype",
		"helm",
		"jq",
		"k9s",
		"kind",
		"ko",
		"kubectl",
		"lazygit",
		"linkerd",
		"nerdctl",
		"nodejs",
		"opentofu",
		"protoc",
		"protoc-gen-go",
		"protoc-gen-go-grpc",
		"protoc-gen-grpc-web",
		"pipx",
		"protolint",
		"python",
		"rust",
		"sccache",
		"shellcheck",
		"shfmt",
		"sops",
		"sqlc",
		"syft",
		"tekton-cli",
		"telepresence",
		"terraform",
		"terragrunt",
		"terrascan",
		"tfupdate",
		"tflint",
		"traefik",
		"trivy",
		"upx",
		"uv",
		"velero",
		"vultr-cli",
		"yq",
		"zig",
	}
}
