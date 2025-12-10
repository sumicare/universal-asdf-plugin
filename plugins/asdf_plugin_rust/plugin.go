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

package asdf_plugin_rust

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
	"sort"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errRustNoChannelFound is returned when a rust-toolchain.toml file has no channel.
	errRustNoChannelFound = errors.New("no channel found in file")
	// errRustNoVersionsMatching is returned when no Rust versions match a LatestStable query.
	errRustNoVersionsMatching = errors.New("no versions matching query")
	// errRustDownloadFailed indicates a non-success HTTP response when downloading rustup.
	errRustDownloadFailed = errors.New("download failed")

	// rustupDownloadURL is the URL used by Download to fetch the rustup installer.
	// It is defined as a variable so tests can override it with a mock server.
	rustupDownloadURL = rustupURL //nolint:gochecknoglobals // we're mutating this for testing, not thread-safe but meh
)

const (
	// rustGitRepoURL points to the upstream Rust GitHub repository.
	rustGitRepoURL = "https://github.com/rust-lang/rust"
	// rustupURL is the URL of the rustup installation script.
	rustupURL = "https://sh.rustup.rs"
)

// Plugin implements the asdf.Plugin interface for Rust.
type Plugin struct {
	githubClient *github.Client
}

// New creates a new Rust plugin instance.
func New() *Plugin {
	return &Plugin{
		githubClient: github.NewClient(),
	}
}

// NewWithClient creates a new Rust plugin with custom client (for testing).
func NewWithClient(client *github.Client) *Plugin {
	return &Plugin{
		githubClient: client,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "rust"
}

// ListBinPaths returns the binary paths for Rust installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for Rust execution.
func (*Plugin) ExecEnv(installPath string) map[string]string {
	return map[string]string{
		"CARGO_HOME":  installPath,
		"RUSTUP_HOME": installPath,
	}
}

// ListLegacyFilenames returns legacy version filenames for Rust.
func (*Plugin) ListLegacyFilenames() []string {
	return []string{"rust-toolchain", "rust-toolchain.toml"}
}

// ParseLegacyFile parses a legacy Rust version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
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
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Rust plugin.
func (*Plugin) Help() asdf.PluginHelp {
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
func (plugin *Plugin) ListAll(ctx context.Context) ([]string, error) {
	tags, err := plugin.githubClient.GetTags(ctx, rustGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	versionRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	versions := make([]string, 0, len(tags))
	for _, tag := range tags {
		if versionRegex.MatchString(tag) {
			versions = append(versions, tag)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return asdf.CompareVersions(versions[i], versions[j]) < 0
	})

	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		versions = stable
	}

	result := []string{"stable", "beta", "nightly"}

	result = append(result, versions...)

	return result, nil
}

// LatestStable returns the latest stable Rust version.
func (plugin *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	if query == "" || query == "stable" {
		return "stable", nil
	}

	if query == "beta" {
		return "beta", nil
	}

	if query == "nightly" {
		return "nightly", nil
	}

	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	// Filter by query
	var filtered []string
	for _, v := range versions {
		if v != "stable" && v != "beta" && v != "nightly" && strings.HasPrefix(v, query) {
			filtered = append(filtered, v)
		}
	}

	if len(filtered) == 0 {
		return "", fmt.Errorf("%w: %s", errRustNoVersionsMatching, query)
	}

	return filtered[len(filtered)-1], nil
}

// Download downloads rustup installer.
func (*Plugin) Download(ctx context.Context, version, downloadPath string) error {
	scriptPath := filepath.Join(downloadPath, "rustup-init.sh")
	if info, err := os.Stat(scriptPath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for rust %s", version)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rustupDownloadURL, http.NoBody)
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

// Install installs Rust using rustup.
func (*Plugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
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

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running rustup-init: %w", err)
	}

	return nil
}
