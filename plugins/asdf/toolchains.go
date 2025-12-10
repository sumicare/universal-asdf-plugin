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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// execCommandContext is a variable for exec.CommandContext to allow mocking in tests.
	execCommandContext = exec.CommandContext //nolint:gochecknoglobals // used for testing
	// execLookPath is a variable for exec.LookPath to allow mocking in tests.
	execLookPath = exec.LookPath //nolint:gochecknoglobals // used for testing
	// osGetwd is a variable for os.Getwd to allow mocking in tests.
	osGetwd = os.Getwd //nolint:gochecknoglobals // used for testing
	// osUserHomeDir is a variable for os.UserHomeDir to allow mocking in tests.
	osUserHomeDir = os.UserHomeDir //nolint:gochecknoglobals // used for testing
)

// EnsureToolchains ensures that the given tools are installed via asdf using
// versions from a .tool-versions file. It prefers a .tool-versions in the
// current working directory and falls back to $HOME/.tool-versions. If the
// asdf binary is not available in PATH, it still ensures that the
// .tool-versions entries exist but skips running `asdf install` so that
// callers can run in environments where asdf has not been bootstrapped yet
// (for example, in CI).
func EnsureToolchains(ctx context.Context, tools ...string) error {
	_ = ctx

	if len(tools) == 0 {
		return nil
	}

	toolVersionsPath, err := resolveToolVersionsPath()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		version := resolveAsdfLatestVersion(ctx, tool)
		if err := ensureToolVersionLine(toolVersionsPath, tool, version); err != nil {
			return err
		}
	}

	return nil
}

// EnsureToolVersionsFile ensures that the given tools have concrete version
// entries in the specified .tool-versions file. It does not run `asdf
// install`; callers are responsible for ensuring the corresponding installs
// exist. For each tool, it attempts to resolve the latest available version
// via `asdf latest <tool>` when the asdf binary is available and falls back
// to "latest" if resolution fails or asdf is not present.
func EnsureToolVersionsFile(ctx context.Context, path string, tools ...string) error {
	_ = ctx

	if len(tools) == 0 {
		return nil
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(path, []byte(""), CommonFilePermission); writeErr != nil {
			return fmt.Errorf("creating %s: %w", path, writeErr)
		}
	}

	for _, tool := range tools {
		version := resolveAsdfLatestVersion(ctx, tool)
		if err := ensureToolVersionLine(path, tool, version); err != nil {
			return err
		}
	}

	return nil
}

// resolveToolVersionsPath returns the path to the .tool-versions file to use
// for installing toolchains. It prefers the current working directory and
// falls back to $HOME/.tool-versions, creating an empty file there if needed.
func resolveToolVersionsPath() (string, error) {
	cwd, err := osGetwd()
	if err == nil {
		p := filepath.Join(cwd, ".tool-versions")
		if _, statErr := os.Stat(p); statErr == nil {
			return p, nil
		}
	}

	home, err := osUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory for .tool-versions: %w", err)
	}

	path := filepath.Join(home, ".tool-versions")
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(path, []byte(""), CommonFilePermission); writeErr != nil {
			return "", fmt.Errorf("creating %s: %w", path, writeErr)
		}
	}

	return path, nil
}

// resolveAsdfLatestVersion attempts to resolve the latest version of the given tool
// using `asdf latest <tool>`. It returns "latest" if asdf is not available or
// if resolution fails.
func resolveAsdfLatestVersion(ctx context.Context, tool string) string {
	path, err := execLookPath("asdf")
	if err != nil {
		return "latest"
	}

	cmd := execCommandContext(ctx, path, "latest", tool)

	out, err := cmd.Output()
	if err != nil {
		return "latest"
	}

	version := strings.TrimSpace(string(out))
	if version == "" {
		return "latest"
	}

	return version
}

// ensureToolVersionLine ensures that the given tool has a version entry in
// the specified .tool-versions file. If missing, it appends `tool <version>`.
func ensureToolVersionLine(path, tool, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		if strings.HasPrefix(line, tool+" ") {
			return nil
		}
	}

	newline := tool + " " + version + "\n"

	data = append(data, []byte(newline)...)
	if err := os.WriteFile(path, data, CommonFilePermission); err != nil {
		return fmt.Errorf("updating %s with %s: %w", path, tool, err)
	}

	return nil
}

// InstallToolchain provides a generic implementation for installing a tool
// using the ASDF directory structure. It resolves ASDF_DATA_DIR, determines
// the latest stable version, creates necessary directories, and calls the
// plugin's Install method.
func InstallToolchain(ctx context.Context, pluginName string, plugin Plugin) error {
	dataDir := os.Getenv("ASDF_DATA_DIR")
	if dataDir == "" {
		home, err := osUserHomeDir()
		if err != nil {
			return fmt.Errorf("determining home directory for ASDF_DATA_DIR fallback: %w", err)
		}

		dataDir = filepath.Join(home, ".asdf")
	}

	version, err := plugin.LatestStable(ctx, "")
	if err != nil || version == "" {
		return fmt.Errorf("determining latest version for %s: %w", pluginName, err)
	}

	installPath := filepath.Join(dataDir, "installs", pluginName, version)
	downloadPath := filepath.Join(dataDir, "downloads", pluginName, version)

	if err := os.MkdirAll(downloadPath, CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating download directory for %s: %w", pluginName, err)
	}

	if err := os.MkdirAll(installPath, CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating install directory for %s: %w", pluginName, err)
	}

	if err := plugin.Install(ctx, version, downloadPath, installPath); err != nil {
		return fmt.Errorf("installing %s %s: %w", pluginName, version, err)
	}

	return nil
}

// EnsureGitRepo ensures that a git repository exists at the specified path.
// If the repository does not exist, it clones it from the given URL with
// --depth 1. If it exists, it attempts to update it with git pull --ff-only
// (best effort, errors ignored). Optional messages can be provided for install
// and success notifications.
func EnsureGitRepo(ctx context.Context, repoPath, gitURL, installMsg, successMsg string) error {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if installMsg != "" {
			Msgf(installMsg)
		}

		if err := os.MkdirAll(filepath.Dir(repoPath), CommonDirectoryPermission); err != nil {
			return fmt.Errorf("creating directory for %s: %w", repoPath, err)
		}

		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", gitURL, repoPath)

		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cloning %s: %w", gitURL, err)
		}

		if successMsg != "" {
			Msgf(successMsg)
		}
	} else if err == nil {
		cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "--ff-only")

		_ = cmd.Run() //nolint:errcheck // we're fine
	} else {
		return fmt.Errorf("checking %s: %w", repoPath, err)
	}

	return nil
}
