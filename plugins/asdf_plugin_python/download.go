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
	"path/filepath"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// Download downloads the specified Python version (no-op for python-build).
func (p *Plugin) Download(ctx context.Context, _, _ string) error {
	return p.ensurePythonBuild(ctx)
}

// DownloadFromFTP downloads Python source from python.org FTP.
// This is an alternative method that doesn't require python-build.
func (p *Plugin) DownloadFromFTP(ctx context.Context, version, downloadPath string) error {
	baseURL := p.ftpURL
	if baseURL != "" && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	downloadURL := fmt.Sprintf("%s/%s/Python-%s.tgz", baseURL, version, version)
	archivePath := filepath.Join(downloadPath, "Python-"+version+".tgz")

	asdf.Msgf("Downloading Python %s from %s", version, downloadURL)

	if err := asdf.DownloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("downloading Python %s: %w", version, err)
	}

	return nil
}
