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

package asdf_plugin_python

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// execCommandContextFnPython wraps exec.CommandContext to allow tests to stub
// external command execution (e.g., ldconfig, pip, tar, configure, make)
// without invoking real system binaries.
//
//nolint:gochecknoglobals // test seam for stubbing external commands in install logic
var execCommandContextFnPython = exec.CommandContext

// Install installs the specified Python version using python-build.
func (plugin *Plugin) Install(ctx context.Context, version, _, installPath string) error {
	if err := plugin.verifyBuildDeps(ctx); err != nil {
		return err
	}

	if err := plugin.ensurePythonBuild(ctx); err != nil {
		return err
	}

	pythonBuildPath := plugin.pythonBuildPath()

	asdf.Msgf("Installing Python %s to %s", version, installPath)

	patchURL := os.Getenv("ASDF_PYTHON_PATCH_URL")
	patchDir := os.Getenv("ASDF_PYTHON_PATCHES_DIRECTORY")

	var cmd *exec.Cmd
	if patchURL != "" {
		asdf.Msgf("Applying patch from %s", patchURL)

		patchData, err := execCommandContextFnPython(ctx, "curl", "-sSL", patchURL).Output()
		if err != nil {
			return fmt.Errorf("downloading patch from %s: %w", patchURL, err)
		}

		cmd = execCommandContextFnPython(ctx, pythonBuildPath, "--patch", version, installPath)
		cmd.Stdin = bytes.NewReader(patchData)
	} else if patchDir != "" {
		patchFile := filepath.Join(patchDir, version+".patch")
		if _, err := os.Stat(patchFile); err == nil {
			asdf.Msgf("Applying patch from %s", patchFile)

			patchReader, err := os.Open(patchFile)
			if err != nil {
				return fmt.Errorf("opening patch file %s: %w", patchFile, err)
			}
			defer patchReader.Close()

			cmd = execCommandContextFnPython(ctx, pythonBuildPath, version, installPath, "-p")
			cmd.Stdin = patchReader
		}
	}

	if cmd == nil {
		cmd = execCommandContextFnPython(ctx, pythonBuildPath, version, installPath)
	}

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing Python %s: %w", version, err)
	}

	if err := plugin.installDefaultPackages(ctx, installPath); err != nil {
		asdf.Errf("Warning: failed to install default packages: %v", err)
	}

	asdf.Msgf("Python %s installed successfully", version)

	return nil
}

// installDefaultPackages installs packages from ~/.default-python-packages.
func (*Plugin) installDefaultPackages(ctx context.Context, installPath string) error {
	defaultPkgsFile := os.Getenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
	if defaultPkgsFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		defaultPkgsFile = filepath.Join(homeDir, ".default-python-packages")
	}

	if _, err := os.Stat(defaultPkgsFile); os.IsNotExist(err) {
		return nil
	}

	asdf.Msgf("Installing default Python packages...")

	pipPath := filepath.Join(installPath, "bin", "pip")
	cmd := execCommandContextFnPython(ctx, pipPath, "install", "-U", "-r", defaultPkgsFile)

	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(installPath, "bin")+":"+os.Getenv("PATH"),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// InstallFromSource compiles Python from source.
// This is an alternative method that doesn't require python-build.
func (plugin *Plugin) InstallFromSource(ctx context.Context, version, downloadPath, installPath string) error {
	if err := plugin.verifyBuildDeps(ctx); err != nil {
		return err
	}

	archivePath := filepath.Join(downloadPath, "Python-"+version+".tgz")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		if err := plugin.DownloadFromFTP(ctx, version, downloadPath); err != nil {
			return err
		}
	}

	asdf.Msgf("Extracting Python %s...", version)

	cmd := execCommandContextFnPython(ctx, "tar", "-xzf", archivePath, "-C", downloadPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	srcDir := filepath.Join(downloadPath, "Python-"+version)

	asdf.Msgf("Configuring Python %s...", version)

	configureCmd := execCommandContextFnPython(ctx, "./configure", "--prefix="+installPath)

	configureCmd.Dir = srcDir
	configureCmd.Stdout = os.Stderr

	configureCmd.Stderr = os.Stderr
	if err := configureCmd.Run(); err != nil {
		return fmt.Errorf("configuring Python: %w", err)
	}

	asdf.Msgf("Building Python %s...", version)

	makeCmd := execCommandContextFnPython(ctx, "make", "-j4")

	makeCmd.Dir = srcDir
	makeCmd.Stdout = os.Stderr

	makeCmd.Stderr = os.Stderr
	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("building Python: %w", err)
	}

	asdf.Msgf("Installing Python %s...", version)

	installCmd := execCommandContextFnPython(ctx, "make", "install")

	installCmd.Dir = srcDir
	installCmd.Stdout = os.Stderr

	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("installing Python: %w", err)
	}

	if err := plugin.installDefaultPackages(ctx, installPath); err != nil {
		asdf.Errf("Warning: failed to install default packages: %v", err)
	}

	asdf.Msgf("Python %s installed successfully", version)

	return nil
}

// ReadDefaultPackages reads the default packages file and returns package names.
func (*Plugin) ReadDefaultPackages() ([]string, error) {
	defaultPkgsFile := os.Getenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
	if defaultPkgsFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		defaultPkgsFile = filepath.Join(homeDir, ".default-python-packages")
	}

	file, err := os.Open(defaultPkgsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}
	defer file.Close()

	var packages []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			packages = append(packages, line)
		}
	}

	return packages, scanner.Err()
}

// verifyBuildDeps verifies that required system libraries for building Python are
// available via the dynamic linker cache. It uses ldconfig -p and fails with a
// descriptive error if libraries are missing, unless ASDF_PYTHON_SKIP_SYSDEPS_CHECK
// is set to a non-empty value.
func (*Plugin) verifyBuildDeps(ctx context.Context) error {
	if os.Getenv("ASDF_PYTHON_SKIP_SYSDEPS_CHECK") != "" {
		return nil
	}

	required := []struct {
		name    string
		feature string
	}{
		{"libbz2.so", "bz2 compression (bz2 module)"},
		{"libreadline.so", "readline support (readline module)"},
		{"libncursesw.so", "terminal UI (curses module)"},
		{"libssl.so", "TLS/SSL (ssl module)"},
		{"libsqlite3.so", "SQLite (sqlite3 module)"},
		{"libgdbm.so", "GDBM database (gdbm module)"},
		{"libffi.so", "FFI (ctypes/ffi modules)"},
		{"libz.so", "zlib compression (zlib/gzip modules)"},
		{"libuuid.so", "UUID support (uuid module)"},
		{"liblzma.so", "XZ compression (lzma module)"},
	}

	cmd := execCommandContextFnPython(ctx, "ldconfig", "-p")

	output, err := cmd.Output()
	if err != nil {
		asdf.Errf("Warning: could not run ldconfig to verify Python build dependencies: %v", err)
		return nil
	}

	found := make(map[string]bool, len(required))

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		for _, lib := range required { //nolint:gocritic // meh let's copy go brr....
			if found[lib.name] {
				continue
			}

			if strings.Contains(line, lib.name) {
				found[lib.name] = true
			}
		}
	}

	var missing []string
	for _, lib := range required { //nolint:gocritic // meh let's copy go brr....
		if !found[lib.name] {
			missing = append(missing, fmt.Sprintf("%s (%s)", lib.name, lib.feature))
		}
	}

	if len(missing) == 0 {
		return nil
	}

	//nolint:err113 // it's probably the last thing the user will see anyway
	return fmt.Errorf(
		"missing system libraries required to build Python: %s."+
			"Set ASDF_PYTHON_SKIP_SYSDEPS_CHECK=1 to override this check (build may succeed but the Python standard library can be incomplete)",
		strings.Join(missing, ", "),
	)
}
