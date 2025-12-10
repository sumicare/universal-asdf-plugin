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
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// AsdfPlugin implements asdf plugin functionality for asdf itself.
type AsdfPlugin struct {
	*asdf.BinaryPlugin
}

// NewAsdfPlugin creates a new asdf plugin instance.
func NewAsdfPlugin() asdf.Plugin {
	return &AsdfPlugin{asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:      "asdf",
		RepoOwner: "asdf-vm",
		RepoName:  "asdf",

		FileNameTemplate: "asdf-v{{.Version}}-{{.Platform}}-{{.Arch}}.tar.gz",
		ArchiveType:      "tar.gz",

		DownloadURLTemplate: "https://github.com/asdf-vm/asdf/releases/download/v{{.Version}}/{{.FileName}}",

		BinaryName: "asdf",

		OsMap: map[string]string{
			"linux":  "linux",
			"darwin": "darwin",
		},
		ArchMap: map[string]string{
			"amd64": "amd64",
			"arm64": "arm64",
			"386":   "386",
		},

		VersionFilter: `^(0\.(1[6-9]|[2-9][0-9])|[1-9][0-9]*\.)`,

		HelpDescription: "Extendable version manager",
		HelpLink:        "https://asdf-vm.com",
	})}
}

// Help returns plugin help information with shell configuration instructions.
func (*AsdfPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `asdf - Extendable version manager

asdf is a CLI tool that manages multiple language runtime versions on a
per-project basis. This plugin enables self-management of asdf, allowing
you to bootstrap asdf using universal-asdf-plugin.

BOOTSTRAP USAGE:
  # Install asdf plugin and bootstrap
  universal-asdf-plugin install-plugin asdf

  # Or add to .tool-versions and install
  echo "asdf 0.18.0" >> .tool-versions
  universal-asdf-plugin install asdf

After installation, configure your shell to use asdf shims.`,

		Config: `SHELL CONFIGURATION:

After installing asdf, you must configure your shell to use asdf shims.
Add the appropriate configuration to your shell's RC file:

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

BASH (~/.bashrc or ~/.bash_profile):

  # Add shims to PATH
  export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

  # Optional: Enable completions
  . <(asdf completion bash)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ZSH (~/.zshrc):

  # Add shims to PATH
  export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

  # Optional: Enable completions
  mkdir -p "${ASDF_DATA_DIR:-$HOME/.asdf}/completions"
  asdf completion zsh > "${ASDF_DATA_DIR:-$HOME/.asdf}/completions/_asdf"
  fpath=(${ASDF_DATA_DIR:-$HOME/.asdf}/completions $fpath)
  autoload -Uz compinit && compinit

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

FISH (~/.config/fish/config.fish):

  # ASDF configuration code
  if test -z $ASDF_DATA_DIR
      set _asdf_shims "$HOME/.asdf/shims"
  else
      set _asdf_shims "$ASDF_DATA_DIR/shims"
  end

  if not contains $_asdf_shims $PATH
      set -gx --prepend PATH $_asdf_shims
  end
  set --erase _asdf_shims

  # Optional: Enable completions
  asdf completion fish > ~/.config/fish/completions/asdf.fish

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

POWERSHELL (~/.config/powershell/profile.ps1):

  # Determine the location of the shims directory
  if ($null -eq $ASDF_DATA_DIR -or $ASDF_DATA_DIR -eq '') {
      $_asdf_shims = "${env:HOME}/.asdf/shims"
  } else {
      $_asdf_shims = "$ASDF_DATA_DIR/shims"
  }
  $env:PATH = "${_asdf_shims}:${env:PATH}"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NUSHELL (~/.config/nushell/config.nu):

  let shims_dir = (
      if ($env | get --ignore-errors ASDF_DATA_DIR | is-empty) {
          $env.HOME | path join '.asdf'
      } else {
          $env.ASDF_DATA_DIR
      } | path join 'shims'
  )
  $env.PATH = (
      $env.PATH | split row (char esep)
      | where { |p| $p != $shims_dir }
      | prepend $shims_dir
  )

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

CUSTOM DATA DIRECTORY (optional):

  To use a custom directory instead of ~/.asdf, set ASDF_DATA_DIR:

  export ASDF_DATA_DIR="/your/custom/data/dir"

  Add this BEFORE the PATH configuration in your shell RC file.`,

		Links: `Homepage:     https://asdf-vm.com
Repository:   https://github.com/asdf-vm/asdf
Releases:     https://github.com/asdf-vm/asdf/releases
Getting Started: https://asdf-vm.com/guide/getting-started.html`,
	}
}

// Install installs asdf to the specified path and creates the shim.
func (plugin *AsdfPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	if err := plugin.BinaryPlugin.Install(ctx, version, downloadPath, installPath); err != nil {
		return err
	}

	plugin.printShellConfigHelp()

	return nil
}

// printShellConfigHelp prints shell configuration instructions.
func (plugin *AsdfPlugin) printShellConfigHelp() {
	shell := plugin.detectShell()

	asdf.Msgf("\n" + strings.Repeat("━", 70))
	asdf.Msgf("asdf installed successfully!")
	asdf.Msgf(strings.Repeat("━", 70))

	switch shell {
	case "bash":
		asdf.Msgf("\nAdd to ~/.bashrc or ~/.bash_profile:")
		asdf.Msgf("  export PATH=\"${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH\"")
		asdf.Msgf("  . <(asdf completion bash)  # optional completions")

	case "zsh":
		asdf.Msgf("\nAdd to ~/.zshrc:")
		asdf.Msgf("  export PATH=\"${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH\"")

	case "fish":
		asdf.Msgf("\nAdd to ~/.config/fish/config.fish:")
		asdf.Msgf("  See 'asdf help' for full Fish configuration")

	default:
		asdf.Msgf("\nAdd asdf shims to your PATH:")
		asdf.Msgf("  export PATH=\"${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH\"")
	}

	asdf.Msgf("\nThen restart your shell or run:")
	asdf.Msgf("  source ~/.%src  # or your shell's RC file", shell)
	asdf.Msgf("\nFor full configuration options, run:")
	asdf.Msgf("  asdf help")
	asdf.Msgf(strings.Repeat("━", 70) + "\n")
}

// detectShell attempts to detect the current shell.
func (*AsdfPlugin) detectShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		base := filepath.Base(shell)
		switch base {
		case "bash", "zsh", "fish", "elvish", "nu", "pwsh":
			return base
		}
	}

	if runtime.GOOS == "windows" {
		return "pwsh"
	}

	return "bash"
}

// GetDataDir returns the asdf data directory.
func (*AsdfPlugin) GetDataDir() string {
	if dir := os.Getenv("ASDF_DATA_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".asdf")
}

// GetShimsDir returns the asdf shims directory.
func (plugin *AsdfPlugin) GetShimsDir() string {
	return filepath.Join(plugin.GetDataDir(), "shims")
}

// GetPluginsDir returns the asdf plugins directory.
func (plugin *AsdfPlugin) GetPluginsDir() string {
	return filepath.Join(plugin.GetDataDir(), "plugins")
}

// IsAsdfInstalled checks if asdf is installed in the data directory.
func (plugin *AsdfPlugin) IsAsdfInstalled() bool {
	shimsDir := plugin.GetShimsDir()
	asdfShim := filepath.Join(shimsDir, "asdf")

	_, err := os.Stat(asdfShim)

	return err == nil
}

// IsAsdfInPath checks if asdf is available in PATH.
func (plugin *AsdfPlugin) IsAsdfInPath() bool {
	path := os.Getenv("PATH")
	shimsDir := plugin.GetShimsDir()

	return slices.Contains(filepath.SplitList(path), shimsDir)
}

// GetShellConfigInstructions returns shell-specific configuration instructions.
func (*AsdfPlugin) GetShellConfigInstructions(shell string) string {
	switch shell {
	case "bash":
		return `Add to ~/.bashrc or ~/.bash_profile:

  export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

  # Optional: Enable completions
  . <(asdf completion bash)`

	case "zsh":
		return `Add to ~/.zshrc:

  export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

  # Optional: Enable completions
  mkdir -p "${ASDF_DATA_DIR:-$HOME/.asdf}/completions"
  asdf completion zsh > "${ASDF_DATA_DIR:-$HOME/.asdf}/completions/_asdf"
  fpath=(${ASDF_DATA_DIR:-$HOME/.asdf}/completions $fpath)
  autoload -Uz compinit && compinit`

	case "fish":
		return `Add to ~/.config/fish/config.fish:

  if test -z $ASDF_DATA_DIR
      set _asdf_shims "$HOME/.asdf/shims"
  else
      set _asdf_shims "$ASDF_DATA_DIR/shims"
  end

  if not contains $_asdf_shims $PATH
      set -gx --prepend PATH $_asdf_shims
  end
  set --erase _asdf_shims

  # Optional: Enable completions
  asdf completion fish > ~/.config/fish/completions/asdf.fish`

	case "pwsh", "powershell":
		return `Add to ~/.config/powershell/profile.ps1:

  if ($null -eq $ASDF_DATA_DIR -or $ASDF_DATA_DIR -eq '') {
      $_asdf_shims = "${env:HOME}/.asdf/shims"
  } else {
      $_asdf_shims = "$ASDF_DATA_DIR/shims"
  }
  $env:PATH = "${_asdf_shims}:${env:PATH}"`

	case "nu", "nushell":
		return `Add to ~/.config/nushell/config.nu:

  let shims_dir = (
      if ($env | get --ignore-errors ASDF_DATA_DIR | is-empty) {
          $env.HOME | path join '.asdf'
      } else {
          $env.ASDF_DATA_DIR
      } | path join 'shims'
  )
  $env.PATH = (
      $env.PATH | split row (char esep)
      | where { |p| $p != $shims_dir }
      | prepend $shims_dir
  )`

	case "elvish":
		return `Add to ~/.config/elvish/rc.elv:

  var asdf_data_dir = ~'/.asdf'
  if (and (has-env ASDF_DATA_DIR) (!=s $E:ASDF_DATA_DIR '')) {
      set asdf_data_dir = $E:ASDF_DATA_DIR
  }
  if (not (has-value $paths $asdf_data_dir'/shims')) {
      set paths = [$asdf_data_dir'/shims' $@paths]
  }`

	default:
		return `Add asdf shims to your PATH:

  export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"`
	}
}
