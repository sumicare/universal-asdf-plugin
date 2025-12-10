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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errGoVersionNotFoundInFile is returned when a go.mod/go.work file has no usable Go version.
	errGoVersionNotFoundInFile = errors.New("no go version found in file")
	// errGoNoVersionsFound is returned when no Go versions are discovered.
	errGoNoVersionsFound = errors.New("no versions found")
)

const (
	// goDownloadURL is the base URL used to download official Go release archives.
	goDownloadURL = "https://dl.google.com/go"
	// goGitRepoURL is the upstream Git repository for the Go project.
	goGitRepoURL = "https://github.com/golang/go"
)

// GolangPlugin implements the asdf.Plugin interface for Go.
type GolangPlugin struct {
	*github.Client
}

// NewGolangPlugin creates a new Go plugin instance.
func NewGolangPlugin() asdf.Plugin {
	return &GolangPlugin{
		github.NewClient(),
	}
}

// Name returns the plugin name.
func (*GolangPlugin) Name() string {
	return "golang"
}

// ListBinPaths returns the binary paths for Go installations.
func (*GolangPlugin) ListBinPaths() string {
	return "go/bin bin"
}

// ExecEnv returns environment variables for Go execution.
// Only sets variables if not already set by user.
func (*GolangPlugin) ExecEnv(installPath string) map[string]string {
	env := make(map[string]string)

	if os.Getenv("GOROOT") == "" {
		env["GOROOT"] = filepath.Join(installPath, "go")
	}

	if os.Getenv("GOPATH") == "" {
		env["GOPATH"] = filepath.Join(installPath, "packages")
	}

	if os.Getenv("GOBIN") == "" {
		env["GOBIN"] = filepath.Join(installPath, "bin")
	}

	return env
}

// ListLegacyFilenames returns legacy version filenames for Go.
func (*GolangPlugin) ListLegacyFilenames() []string {
	return []string{".go-version", "go.mod", "go.work"}
}

// ParseLegacyFile parses a legacy Go version file.
// Supports .go-version, go.mod, and go.work files.
func (*GolangPlugin) ParseLegacyFile(path string) (string, error) {
	basename := filepath.Base(path)

	if basename == "go.mod" || basename == "go.work" {
		return parseGoModVersion(path)
	}

	version, err := asdf.ReadLegacyVersionFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(version, "go"), nil
}

// parseGoModVersion extracts the Go version from go.mod or go.work files.
func parseGoModVersion(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.SplitSeq(string(content), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "heroku goVersion") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part != "goVersion" || i+1 >= len(parts) {
					continue
				}

				version := strings.TrimPrefix(parts[i+1], "go")

				return version, nil
			}
		}

		if !strings.HasPrefix(line, "go ") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		version := parts[1]

		if version == "" || version[0] < '0' || version[0] > '9' {
			continue
		}

		return version, nil
	}

	return "", fmt.Errorf("%w: %s", errGoVersionNotFoundInFile, path)
}

// Uninstall removes a Go installation.
func (*GolangPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Go plugin.
func (*GolangPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Go (golang) - An open-source programming language supported by Google.
This plugin downloads pre-built Go binaries from https://go.dev/dl/`,
		Deps: `No system dependencies required - uses pre-built binaries.`,
		Config: `Environment variables:
  ASDF_GOLANG_DEFAULT_PACKAGES_FILE - Path to default packages file (default: ~/.default-golang-pkgs)
  ASDF_GOLANG_MOD_VERSION_ENABLED - Enable go.mod version detection (default: false)`,
		Links: `Homepage: https://go.dev/
Documentation: https://go.dev/doc/
Downloads: https://go.dev/dl/
Source: https://github.com/golang/go`,
	}
}

// ListAll returns all available Go versions using the GitHub API.
func (p *GolangPlugin) ListAll(ctx context.Context) ([]string, error) {
	tags, err := p.GetTags(ctx, goGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("listing go versions: %w", err)
	}

	versions := parseGoTags(tags)

	versions = filterOldVersions(versions)

	// Prefer stable versions in list-all output when possible, but keep
	// prereleases when no stable versions exist.
	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		versions = stable
	}

	sortGoVersions(versions)

	return versions, nil
}

// LatestStable returns the latest stable Go version.
func (p *GolangPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := p.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errGoNoVersionsFound
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
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) == 0 {
		return versions[len(versions)-1], nil
	}

	return stable[len(stable)-1], nil
}

// Download downloads the specified Go version.
func (*GolangPlugin) Download(ctx context.Context, version, downloadPath string) error {
	platform, err := asdf.GetPlatform()
	if err != nil {
		return err
	}

	arch, err := asdf.GetArch()
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%s/go%s.%s-%s.tar.gz", goDownloadURL, version, platform, arch)
	archivePath := filepath.Join(downloadPath, "archive.tar.gz")

	asdf.Msgf("Downloading Go %s from %s", version, downloadURL)

	if err := asdf.DownloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("downloading Go %s: %w", version, err)
	}

	if os.Getenv("ASDF_GOLANG_SKIP_CHECKSUM") == "" {
		checksumURL := downloadURL + ".sha256"
		checksumPath := archivePath + ".sha256"

		asdf.Msgf("Downloading checksum from %s", checksumURL)

		if err := asdf.DownloadFile(ctx, checksumURL, checksumPath); err != nil {
			return fmt.Errorf("downloading checksum: %w", err)
		}

		checksumData, err := os.ReadFile(checksumPath)
		if err != nil {
			return fmt.Errorf("reading checksum file: %w", err)
		}

		asdf.Msgf("Verifying checksum...")

		if err := asdf.VerifySHA256(archivePath, string(checksumData)); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}

		asdf.Msgf("Checksum verified")
	} else {
		asdf.Errf("Checksum verification skipped")
	}

	return nil
}

// Install installs Go from the downloaded archive.
func (plugin *GolangPlugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	archivePath := filepath.Join(downloadPath, "archive.tar.gz")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		if err := plugin.Download(ctx, version, downloadPath); err != nil {
			return err
		}
	}

	asdf.Msgf("Installing Go %s to %s", version, installPath)

	if err := asdf.EnsureDir(installPath); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	if err := asdf.ExtractTarGz(archivePath, installPath); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	if err := plugin.installDefaultPackages(ctx, version, installPath); err != nil {
		asdf.Errf("Warning: failed to install default packages: %v", err)
	}

	asdf.Msgf("Go %s installed successfully", version)

	return nil
}

// installDefaultPackages installs packages from ~/.default-golang-pkgs.
func (*GolangPlugin) installDefaultPackages(ctx context.Context, version, installPath string) error {
	defaultPkgsFile := os.Getenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE")
	if defaultPkgsFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil
		}

		defaultPkgsFile = filepath.Join(homeDir, ".default-golang-pkgs")
	}

	if _, err := os.Stat(defaultPkgsFile); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(defaultPkgsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	goBin := filepath.Join(installPath, "go", "bin", "go")
	goRoot := filepath.Join(installPath, "go")
	goPath := filepath.Join(installPath, "packages")
	goBinDir := filepath.Join(installPath, "bin")

	parts := asdf.ParseVersionParts(version)
	useInstall := len(parts) >= 2 && (parts[0] >= 2 || (parts[0] == 1 && parts[1] >= 16))

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		if line == "" {
			continue
		}

		asdf.Msgf("Installing %s...", line)

		var cmd *exec.Cmd
		if useInstall {
			pkg := line
			if !strings.Contains(pkg, "@") {
				pkg += "@latest"
			}

			cmd = exec.CommandContext(ctx, goBin, "install", pkg)
		} else {
			cmd = exec.CommandContext(ctx, goBin, "get", "-u", line)
		}

		cmd.Env = append(os.Environ(),
			"GOROOT="+goRoot,
			"GOPATH="+goPath,
			"GOBIN="+goBinDir,
			"PATH="+filepath.Join(goRoot, "bin")+":"+os.Getenv("PATH"),
		)

		if err := cmd.Run(); err != nil {
			asdf.Errf("Failed to install %s: %v", line, err)
		} else {
			asdf.Msgf("Successfully installed %s", line)
		}
	}

	return scanner.Err()
}

// parseGoTags extracts version numbers from GitHub tag names.
func parseGoTags(tags []string) []string {
	var versions []string
	for _, tag := range tags {
		if after, ok := strings.CutPrefix(tag, "go"); ok {
			versions = append(versions, after)
		}
	}

	return versions
}

// filterOldVersions removes old/unsupported Go versions.
// Filters out: 1, 1.0, 1.0.x, 1.1, 1.1rc*, 1.1.x, 1.2, 1.2rc*, 1.2.1, 1.8.5rc5.
func filterOldVersions(versions []string) []string {
	excludePatterns := []string{
		`^1$`,
		`^1\.0$`,
		`^1\.0\.[0-9]+$`,
		`^1\.1$`,
		`^1\.1rc[0-9]+$`,
		`^1\.1\.[0-9]+$`,
		`^1\.2$`,
		`^1\.2rc[0-9]+$`,
		`^1\.2\.1$`,
		`^1\.8\.5rc5$`,
	}

	compiled := make([]*regexp.Regexp, 0, len(excludePatterns))
	for _, pattern := range excludePatterns {
		compiled = append(compiled, regexp.MustCompile(pattern))
	}

	return asdf.FilterVersions(versions, func(v string) bool {
		for _, re := range compiled {
			if re.MatchString(v) {
				return false
			}
		}

		return true
	})
}

// sortGoVersions sorts versions in semver order.
func sortGoVersions(versions []string) {
	asdf.SortVersions(versions)
}

// EnsureGoToolchainEntries ensures that a golang entry exists in .tool-versions
// via the generic toolchains helper.
func EnsureGoToolchainEntries(ctx context.Context) error {
	return asdf.EnsureToolchains(ctx, "golang")
}

// InstallGoToolchain installs the Go toolchain into an asdf-style tree under
// ASDF_DATA_DIR (or $HOME/.asdf if unset) using the Go plugin implementation.
func InstallGoToolchain(ctx context.Context) error {
	return asdf.InstallToolchain(ctx, "golang", NewGolangPlugin())
}
