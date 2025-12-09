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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// errNodeChecksumNotFound is returned when the expected checksum entry cannot be found.
	errNodeChecksumNotFound = errors.New("checksum not found")

	// getArchFnNode wraps asdf.GetArch so tests can override the reported architecture
	// without mutating global asdf state.
	getArchFnNode = asdf.GetArch //nolint:gochecknoglobals // configurable in tests
)

// Download downloads the specified Node.js version.
func (plugin *Plugin) Download(ctx context.Context, version, downloadPath string) error {
	platform, err := asdf.GetPlatform()
	if err != nil {
		return err
	}

	arch, err := getNodeArch()
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%sv%s/node-v%s-%s-%s.tar.gz", plugin.distURL, version, version, platform, arch)
	archivePath := filepath.Join(downloadPath, "node.tar.gz")

	asdf.Msgf("Downloading Node.js %s from %s", version, downloadURL)

	if err := asdf.DownloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("downloading Node.js %s: %w", version, err)
	}

	shasumsURL := fmt.Sprintf("%sv%s/SHASUMS256.txt", plugin.distURL, version)
	shasumsPath := filepath.Join(downloadPath, "SHASUMS256.txt")

	if err := asdf.DownloadFile(ctx, shasumsURL, shasumsPath); err != nil {
		asdf.Errf("Warning: could not download checksums: %v", err)
	} else {
		expectedFilename := fmt.Sprintf("node-v%s-%s-%s.tar.gz", version, platform, arch)
		if err := verifyNodeChecksum(archivePath, shasumsPath, expectedFilename); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}

		asdf.Msgf("Checksum verified")
	}

	return nil
}

// getNodeArch returns the architecture string for Node.js downloads.
func getNodeArch() (string, error) {
	arch, err := getArchFnNode()
	if err != nil {
		return "", err
	}

	switch arch {
	case "amd64":
		return "x64", nil
	case "386":
		return "x86", nil
	case "arm64":
		return "arm64", nil
	case "armv6l":
		return "armv7l", nil
	default:
		return arch, nil
	}
}

// verifyNodeChecksum verifies the checksum of a Node.js download.
func verifyNodeChecksum(archivePath, shasumsPath, expectedFilename string) error {
	file, err := os.Open(shasumsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasSuffix(parts[1], expectedFilename) {
			return asdf.VerifySHA256(archivePath, parts[0])
		}
	}

	return fmt.Errorf("%w: %s", errNodeChecksumNotFound, expectedFilename)
}
