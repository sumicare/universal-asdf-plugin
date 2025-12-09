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
	"fmt"
	"os"
	"path/filepath"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// ensureToolchainsFnGo is a test seam for the shared EnsureToolchains helper.
	ensureToolchainsFnGo = asdf.EnsureToolchains //nolint:gochecknoglobals // test seams configurable in tests
	// userHomeDirFnGo is a test seam for discovering the current user's home directory.
	userHomeDirFnGo = os.UserHomeDir //nolint:gochecknoglobals // test seams configurable in tests
	// mkdirAllFnGo is a test seam for creating directories.
	mkdirAllFnGo = os.MkdirAll //nolint:gochecknoglobals // test seams configurable in tests
)

// EnsureGoToolchainEntries ensures that a golang entry exists in .tool-versions
// via the generic toolchains helper.
func EnsureGoToolchainEntries(ctx context.Context) error {
	return ensureToolchainsFnGo(ctx, "golang")
}

// InstallGoToolchain installs the Go toolchain into an asdf-style tree under
// ASDF_DATA_DIR (or $HOME/.asdf if unset) using the Go plugin implementation.
func InstallGoToolchain(ctx context.Context) error {
	dataDir := os.Getenv("ASDF_DATA_DIR")
	if dataDir == "" {
		home, err := userHomeDirFnGo()
		if err != nil {
			return fmt.Errorf("determining home directory for ASDF_DATA_DIR fallback: %w", err)
		}

		dataDir = filepath.Join(home, ".asdf")
	}

	plug := New()

	version, err := plug.LatestStable(ctx, "")
	if err != nil || version == "" {
		return fmt.Errorf("determining latest version for golang: %w", err)
	}

	installPath := filepath.Join(dataDir, "installs", "golang", version)
	downloadPath := filepath.Join(dataDir, "downloads", "golang", version)

	if err := mkdirAllFnGo(downloadPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating download directory for golang: %w", err)
	}

	if err := mkdirAllFnGo(installPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating install directory for golang: %w", err)
	}

	if err := plug.Install(ctx, version, downloadPath, installPath); err != nil {
		return fmt.Errorf("installing golang %s: %w", version, err)
	}

	return nil
}
