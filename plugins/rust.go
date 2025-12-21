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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// errRustNoChannelFound is returned when a rust-toolchain.toml file has no channel.
	errRustNoChannelFound = errors.New("no channel found in file")
	// errRustDownloadFailed indicates a non-success HTTP response when downloading rustup.
	errRustDownloadFailed = errors.New("download failed")
)

const (
	// rustupURL is the URL of the rustup installation script.
	rustupURL = "https://sh.rustup.rs"
)

// RustPlugin implements the asdf.Plugin interface for Rust.
type RustPlugin struct {
	*asdf.SourceBuildPlugin
}

// NewRustPlugin creates a new Rust plugin instance.
func NewRustPlugin() asdf.Plugin {
	plugin := &RustPlugin{}

	plugin.SourceBuildPlugin = asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name:              "rust",
		RepoOwner:         "rust-lang",
		RepoName:          "rust",
		VersionPrefix:     "",
		VersionFilter:     `^\d+\.\d+\.\d+$`,
		BinDir:            "bin",
		SkipDownload:      true,
		SkipExtract:       true,
		ExpectedArtifacts: []string{"bin/rustc", "bin/cargo"},
		LegacyFilenames:   []string{"rust-toolchain", "rust-toolchain.toml"},
		BuildVersion: func(ctx context.Context, version, sourceDir, installPath string) error {
			// Ensure rustup is downloaded
			downloadPath := sourceDir
			if downloadPath == "" {
				downloadPath = filepath.Join(installPath, ".download")

				err := os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)
				if err != nil {
					return fmt.Errorf("creating download directory: %w", err)
				}
			}

			err := plugin.Download(ctx, version, downloadPath)
			if err != nil {
				return err
			}

			scriptPath := filepath.Join(downloadPath, "rustup-init.sh")

			profile := os.Getenv("ASDF_RUST_PROFILE")
			if profile == "" {
				profile = "default"
			}

			env := os.Environ()

			env = append(env, "CARGO_HOME="+installPath, "RUSTUP_HOME="+installPath)

			cmd := exec.CommandContext(ctx, "sh", scriptPath,
				"-y",
				"--no-modify-path",
				"--default-toolchain", version,
				"--profile", profile,
			)

			cmd.Env = env
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("running rustup-init: %w", err)
			}

			// Clean up download directory if we created it
			if sourceDir == "" {
				_ = os.RemoveAll(downloadPath)
			}

			return nil
		},
	})

	return plugin
}

// Name returns the plugin name.
func (*RustPlugin) Name() string {
	return "rust"
}

// ListBinPaths returns the binary paths for Rust installations.
func (*RustPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Rust execution.
func (*RustPlugin) ExecEnv(installPath string) map[string]string {
	return map[string]string{
		"CARGO_HOME":  installPath,
		"RUSTUP_HOME": installPath,
	}
}

// ListLegacyFilenames returns legacy version filenames for Rust.
func (plugin *RustPlugin) ListLegacyFilenames() []string {
	return plugin.SourceBuildPlugin.ListLegacyFilenames()
}

// ParseLegacyFile parses a legacy Rust version file.
func (*RustPlugin) ParseLegacyFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(path, ".toml") {
		re := regexp.MustCompile(`channel\s*=\s*"([^"]+)"`)

		matches := re.FindStringSubmatch(string(content))
		if len(matches) >= 2 {
			return matches[1], nil
		}

		return "", fmt.Errorf("%w: %s", errRustNoChannelFound, path)
	}

	version := strings.TrimSpace(string(content))

	return version, nil
}

// Uninstall removes a Rust installation.
func (plugin *RustPlugin) Uninstall(ctx context.Context, installPath string) error {
	return plugin.SourceBuildPlugin.Uninstall(ctx, installPath)
}

// Help returns help information for the Rust plugin.
func (*RustPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Rust - A language empowering everyone to build reliable and efficient software.
This plugin uses rustup to install Rust toolchains.`,
		Deps: `Requires curl and a C compiler (gcc/clang) for some crates.`,
		Config: `Environment variables:
  ASDF_RUST_PROFILE - Rustup profile to use (default, minimal, complete)
  ASDF_CRATE_DEFAULT_PACKAGES_FILE - Path to default cargo crates file`,
		Links: `Homepage: https://www.rust-lang.org/
Documentation: https://doc.rust-lang.org/
Rustup: https://rustup.rs/
Source: https://github.com/rust-lang/rust`,
	}
}

// ListAll lists all available Rust versions.
func (plugin *RustPlugin) ListAll(ctx context.Context) ([]string, error) {
	versions, err := plugin.SourceBuildPlugin.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Prepend Rust channels
	result := []string{"stable", "beta", "nightly"}

	result = append(result, versions...)

	return result, nil
}

// LatestStable returns the latest stable Rust version.
func (plugin *RustPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	if query == "" || query == "stable" {
		return "stable", nil
	}

	if query == "beta" {
		return "beta", nil
	}

	if query == "nightly" {
		return "nightly", nil
	}

	return plugin.SourceBuildPlugin.LatestStable(ctx, query)
}

// Download downloads rustup installer.
func (*RustPlugin) Download(ctx context.Context, version, downloadPath string) error {
	scriptPath := filepath.Join(downloadPath, "rustup-init.sh")
	if info, err := os.Stat(scriptPath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for rust %s", version)

		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rustupURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading rustup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", errRustDownloadFailed, resp.StatusCode)
	}

	outFile, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if err := os.Chmod(scriptPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("making script executable: %w", err)
	}

	versionFile := filepath.Join(downloadPath, ".rust-version")
	if err := os.WriteFile(versionFile, []byte(version), asdf.CommonFilePermission); err != nil {
		return fmt.Errorf("writing version file: %w", err)
	}

	return nil
}

// Install installs Rust using rustup via SourceBuildPlugin.
func (p *RustPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	return p.SourceBuildPlugin.Install(ctx, version, downloadPath, installPath)
}
