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
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// Install installs Go from the downloaded archive.
func (plugin *Plugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
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
func (*Plugin) installDefaultPackages(ctx context.Context, version, installPath string) error {
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
