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

package asdf_plugin_nodejs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_python"
)

// Install installs Node.js from the downloaded archive.
func (plugin *Plugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	if err := asdf.EnsureToolchains(ctx, "python"); err != nil {
		return err
	}

	if err := installPythonToolchain(ctx); err != nil {
		return err
	}

	archivePath := filepath.Join(downloadPath, "node.tar.gz")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		if err := plugin.Download(ctx, version, downloadPath); err != nil {
			return err
		}
	}

	asdf.Msgf("Installing Node.js %s to %s", version, installPath)

	if err := asdf.EnsureDir(installPath); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	if err := asdf.ExtractTarGz(archivePath, installPath); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	platform, err := asdf.GetPlatform()
	if err != nil {
		return fmt.Errorf("getting platform: %w", err)
	}

	arch, err := getNodeArch()
	if err != nil {
		return fmt.Errorf("getting node architecture: %w", err)
	}

	extractedDir := filepath.Join(installPath, fmt.Sprintf("node-v%s-%s-%s", version, platform, arch))

	if _, err := os.Stat(extractedDir); err == nil {
		entries, err := os.ReadDir(extractedDir)
		if err != nil {
			return fmt.Errorf("reading extracted directory: %w", err)
		}

		for _, entry := range entries {
			src := filepath.Join(extractedDir, entry.Name())

			dst := filepath.Join(installPath, entry.Name())
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("moving %s: %w", entry.Name(), err)
			}
		}

		os.Remove(extractedDir)
	}

	if err := plugin.installDefaultPackages(ctx, installPath); err != nil {
		asdf.Errf("Warning: failed to install default packages: %v", err)
	}

	if os.Getenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK") != "" {
		if err := plugin.enableCorepack(ctx, installPath); err != nil {
			asdf.Errf("Warning: failed to enable corepack: %v", err)
		}
	}

	asdf.Msgf("Node.js %s installed successfully", version)

	return nil
}

// installDefaultPackages installs packages from ~/.default-npm-packages.
func (*Plugin) installDefaultPackages(ctx context.Context, installPath string) error {
	defaultPkgsFile := os.Getenv("ASDF_NPM_DEFAULT_PACKAGES_FILE")
	if defaultPkgsFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil
		}

		defaultPkgsFile = filepath.Join(homeDir, ".default-npm-packages")
	}

	if _, err := os.Stat(defaultPkgsFile); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(defaultPkgsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	npmPath := filepath.Join(installPath, "bin", "npm")
	nodePath := filepath.Join(installPath, "bin", "node")

	var packages []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, " -") || strings.HasPrefix(line, "-") {
			args := strings.Fields(line)
			asdf.Msgf("Running: npm install -g %s", line)

			cmd := exec.CommandContext(ctx, npmPath, append([]string{"install", "-g"}, args...)...)

			cmd.Env = append(os.Environ(),
				"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
			)
			cmd.Stdout = os.Stderr

			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				asdf.Errf("Failed to install %s: %v", line, err)
			}

			continue
		}

		packages = append(packages, strings.Fields(line)...)
	}

	if len(packages) > 0 {
		asdf.Msgf("Running: npm install -g %s", strings.Join(packages, " "))

		cmd := exec.CommandContext(ctx, npmPath, append([]string{"install", "-g"}, packages...)...)

		cmd.Env = append(os.Environ(),
			"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
			"NODE="+nodePath,
		)
		cmd.Stdout = os.Stderr

		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			asdf.Errf("Failed to install packages: %v", err)
		}
	}

	return scanner.Err()
}

// enableCorepack enables corepack for this Node.js installation.
func (*Plugin) enableCorepack(ctx context.Context, installPath string) error {
	corepackPath := filepath.Join(installPath, "bin", "corepack")
	if _, err := os.Stat(corepackPath); os.IsNotExist(err) {
		return nil
	}

	asdf.Msgf("Enabling corepack...")

	cmd := exec.CommandContext(ctx, corepackPath, "enable")

	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// installPythonToolchain installs the Python toolchain into an asdf-style
// installs tree under ASDF_DATA_DIR or ~/.asdf using the shared Python plugin
// helper. It remains a variable so tests can replace it with a fast stub.
var installPythonToolchain = asdf_plugin_python.InstallPythonToolchain //nolint:gochecknoglobals // configurable in tests
