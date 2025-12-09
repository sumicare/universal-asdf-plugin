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

package asdf_plugin_go

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// errGoVersionNotFoundInFile is returned when a go.mod/go.work file has no usable Go version.
var errGoVersionNotFoundInFile = errors.New("no go version found in file")

const (
	// goDownloadURL is the base URL used to download official Go release archives.
	goDownloadURL = "https://dl.google.com/go"
	// goGitRepoURL is the upstream Git repository for the Go project.
	goGitRepoURL = "https://github.com/golang/go"
)

// Plugin implements the asdf.Plugin interface for Go.
type Plugin struct {
	githubClient *github.Client
	downloadURL  string
}

// New creates a new Go plugin instance.
func New() *Plugin {
	return &Plugin{
		downloadURL:  goDownloadURL,
		githubClient: github.NewClient(),
	}
}

// NewWithURLs creates a new Go plugin with custom URLs (for testing).
func NewWithURLs(downloadURL string, githubClient *github.Client) *Plugin {
	return &Plugin{
		downloadURL:  downloadURL,
		githubClient: githubClient,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "golang"
}

// ListBinPaths returns the binary paths for Go installations.
func (*Plugin) ListBinPaths() string {
	return "go/bin bin"
}

// ExecEnv returns environment variables for Go execution.
// Only sets variables if not already set by user.
func (*Plugin) ExecEnv(installPath string) map[string]string {
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
func (*Plugin) ListLegacyFilenames() []string {
	return []string{".go-version", "go.mod", "go.work"}
}

// ParseLegacyFile parses a legacy Go version file.
// Supports .go-version, go.mod, and go.work files.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
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
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the Go plugin.
func (*Plugin) Help() asdf.PluginHelp {
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
