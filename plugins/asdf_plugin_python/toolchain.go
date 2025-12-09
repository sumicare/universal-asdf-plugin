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
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// userHomeDirFnPython is a test seam for discovering the current user's home directory.
	userHomeDirFnPython = os.UserHomeDir //nolint:gochecknoglobals // test seams configurable in tests
	// mkdirAllFnPython is a test seam for creating directories.
	mkdirAllFnPython = os.MkdirAll //nolint:gochecknoglobals // test seams configurable in tests
	// newPythonPluginFnPython is a test seam for constructing the Python plugin.
	newPythonPluginFnPython = New //nolint:gochecknoglobals // test seams configurable in tests
	// pythonLatestStableFn is a test seam for resolving the latest stable Python version.
	pythonLatestStableFn = func(p *Plugin, ctx context.Context, query string) (string, error) { //nolint:gochecknoglobals // test seams configurable in tests
		return p.LatestStable(ctx, query)
	}
	// pythonInstallFn is a test seam for installing Python.
	pythonInstallFn = func(p *Plugin, ctx context.Context, version, downloadPath, installPath string) error { //nolint:gochecknoglobals // test seams configurable in tests
		return p.Install(ctx, version, downloadPath, installPath)
	}
)

// InstallPythonToolchain installs the Python toolchain into an asdf-style tree under
// ASDF_DATA_DIR (or $HOME/.asdf if unset) using the Python plugin implementation.
func InstallPythonToolchain(ctx context.Context) error {
	dataDir := os.Getenv("ASDF_DATA_DIR")
	if dataDir == "" {
		home, err := userHomeDirFnPython()
		if err != nil {
			return fmt.Errorf("determining home directory for ASDF_DATA_DIR fallback: %w", err)
		}

		dataDir = filepath.Join(home, ".asdf")
	}

	plug := newPythonPluginFnPython()

	version, err := pythonLatestStableFn(plug, ctx, "")
	if err != nil || version == "" {
		return fmt.Errorf("determining latest version for python: %w", err)
	}

	installPath := filepath.Join(dataDir, "installs", "python", version)
	downloadPath := filepath.Join(dataDir, "downloads", "python", version)

	if err := mkdirAllFnPython(downloadPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating download directory for python: %w", err)
	}

	if err := mkdirAllFnPython(installPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating install directory for python: %w", err)
	}

	if err := pythonInstallFn(plug, ctx, version, downloadPath, installPath); err != nil {
		return fmt.Errorf("installing python %s: %w", version, err)
	}

	return nil
}
