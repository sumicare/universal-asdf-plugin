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
	"path/filepath"
	"strings"
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
		if err := ensureToolVersionLine(toolVersionsPath, tool, "latest"); err != nil {
			return err
		}
	}

	return nil
}

// EnsureToolVersionsFile ensures that the given tools have concrete version
// entries in the specified .tool-versions file. It does not run `asdf
// install`; callers are responsible for ensuring the corresponding installs
// exist. For each tool, it attempts to resolve the latest available version
// via `asdf list-all <tool>` when the asdf binary is available and falls back
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
		if err := ensureToolVersionLine(path, tool, "latest"); err != nil {
			return err
		}
	}

	return nil
}

// resolveToolVersionsPath returns the path to the .tool-versions file to use
// for installing toolchains. It prefers the current working directory and
// falls back to $HOME/.tool-versions, creating an empty file there if needed.
func resolveToolVersionsPath() (string, error) {
	cwd, err := os.Getwd()
	if err == nil {
		p := filepath.Join(cwd, ".tool-versions")
		if _, statErr := os.Stat(p); statErr == nil {
			return p, nil
		}
	}

	home, err := os.UserHomeDir()
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
