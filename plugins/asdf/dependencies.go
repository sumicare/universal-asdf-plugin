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

// InstallWithDependencies installs a tool and its dependencies.
// It resolves ASDF_DATA_DIR, determines the latest stable version,
// creates necessary directories, and calls the plugin's Install method.
//
// If the plugin implements PluginWithDependencies, it will automatically
// ensure that all dependencies are present in .tool-versions (resolving versions
// from the project configuration) and installed before proceeding with the
// plugin installation.
func InstallWithDependencies(ctx context.Context, pluginName string, plugin Plugin) error {
	// Handle dependencies if the plugin declares them
	if depsPlugin, ok := plugin.(PluginWithDependencies); ok {
		dependencies := depsPlugin.Dependencies()
		if len(dependencies) > 0 {
			err := installDependencies(ctx, dependencies...)
			if err != nil {
				return fmt.Errorf("installing dependencies for %s: %w", pluginName, err)
			}
		}
	}

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

// installDependencies ensures that the given tools are installed via asdf using
// versions from a .tool-versions file. It prefers a .tool-versions in the
// current working directory and falls back to $HOME/.tool-versions.
func installDependencies(ctx context.Context, tools ...string) error {
	if len(tools) == 0 {
		return nil
	}

	toolVersionsPath, err := ResolveToolVersionsPath()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		version := resolveVersionFromProjectToolVersions(tool)

		err := ensureToolVersionLine(toolVersionsPath, tool, version)
		if err != nil {
			return err
		}
	}

	var asdfPath string
	if path, err := execLookPath("asdf"); err == nil {
		asdfPath = path
	} else {
		return nil
	}

	for _, tool := range tools {
		cmd := execCommandContext(ctx, asdfPath, "install", tool)

		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("running asdf install %s: %w", tool, err)
		}
	}

	return nil
}

// EnsureToolVersionsFile ensures that the given tools have concrete version
// entries in the specified .tool-versions file and installs them in that
// directory context. For each tool, it attempts to resolve the latest stable
// version from the project's .tool-versions (falling back to "latest").
// After ensuring entries exist, it runs `asdf install` in the directory
// containing the .tool-versions file.
func EnsureToolVersionsFile(ctx context.Context, path string, tools ...string) error {
	if len(tools) == 0 {
		return nil
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		writeErr := os.WriteFile(path, []byte(""), CommonFilePermission)
		if writeErr != nil {
			return fmt.Errorf("creating %s: %w", path, writeErr)
		}
	}

	for _, tool := range tools {
		version := resolveVersionFromProjectToolVersions(tool)

		err := ensureToolVersionLine(path, tool, version)
		if err != nil {
			return err
		}
	}

	var asdfPath string
	if path, err := execLookPath("asdf"); err == nil {
		asdfPath = path
	} else {
		return nil
	}

	dirPath := filepath.Dir(path)

	for _, tool := range tools {
		cmd := execCommandContext(ctx, asdfPath, "install", tool)

		cmd.Dir = dirPath
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("running asdf install %s in %s: %w", tool, dirPath, err)
		}
	}

	return nil
}

// ResolveToolVersionsPath returns the path to the .tool-versions file to use
// for installing toolchains. It prefers the current working directory and
// falls back to $HOME/.tool-versions, creating an empty file there if needed.
func ResolveToolVersionsPath() (string, error) {
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
		writeErr := os.WriteFile(path, []byte(""), CommonFilePermission)
		if writeErr != nil {
			return "", fmt.Errorf("creating %s: %w", path, writeErr)
		}
	}

	return path, nil
}

// resolveVersionFromProjectToolVersions reads the version for a tool from the
// project's .tool-versions file. It returns "latest" if the file or tool entry
// is not found.
func resolveVersionFromProjectToolVersions(tool string) string {
	toolVersionsPath, err := ResolveToolVersionsPath()
	if err != nil {
		return "latest"
	}

	data, err := os.ReadFile(toolVersionsPath)
	if err != nil {
		return "latest"
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		if !strings.HasPrefix(line, tool+" ") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	return "latest"
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

		err := os.MkdirAll(filepath.Dir(repoPath), CommonDirectoryPermission)
		if err != nil {
			return fmt.Errorf("creating directory for %s: %w", repoPath, err)
		}

		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", gitURL, repoPath)

		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("cloning %s: %w", gitURL, err)
		}

		if successMsg != "" {
			Msgf(successMsg)
		}
	} else if err == nil {
		cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "--ff-only")

		_ = cmd.Run()
	} else {
		return fmt.Errorf("checking %s: %w", repoPath, err)
	}

	return nil
}
