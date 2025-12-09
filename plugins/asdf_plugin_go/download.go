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

// Download downloads the specified Go version.
func (plugin *Plugin) Download(ctx context.Context, version, downloadPath string) error {
	platform, err := asdf.GetPlatform()
	if err != nil {
		return err
	}

	arch, err := asdf.GetArch()
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%s/go%s.%s-%s.tar.gz", plugin.downloadURL, version, platform, arch)
	archivePath := filepath.Join(downloadPath, "archive.tar.gz")

	asdf.Msgf("Downloading Go %s from %s", version, downloadURL)

	if err := asdf.DownloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("downloading Go %s: %w", version, err)
	}

	if os.Getenv("ASDF_GOLANG_SKIP_CHECKSUM") == "" {
		checksumURL := downloadURL + ".sha256"
		checksumPath := archivePath + ".sha256"

		asdf.Msgf("Downloading checksum from %s", checksumURL)

		if err := asdf.DownloadFile(ctx, checksumURL, checksumPath); err != nil {
			return fmt.Errorf("downloading checksum: %w", err)
		}

		checksumData, err := os.ReadFile(checksumPath)
		if err != nil {
			return fmt.Errorf("reading checksum file: %w", err)
		}

		asdf.Msgf("Verifying checksum...")

		if err := asdf.VerifySHA256(archivePath, string(checksumData)); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}

		asdf.Msgf("Checksum verified")
	} else {
		asdf.Errf("Checksum verification skipped")
	}

	return nil
}
