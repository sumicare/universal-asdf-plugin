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

package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var (
	// errGcloudNoVersionsFound is returned when no gcloud versions are discovered.
	errGcloudNoVersionsFound = errors.New("no versions found")
	// errGcloudNoVersionsMatching is returned when no versions match a LatestStable query.
	errGcloudNoVersionsMatching = errors.New("no versions matching query")
	// errGcloudUnsupportedArch is returned when the current CPU architecture is not supported.
	errGcloudUnsupportedArch = errors.New("unsupported architecture")
	// errGcloudUnsupportedPlatform is returned when the current OS is not supported.
	errGcloudUnsupportedPlatform = errors.New("unsupported platform")
	// errGcloudDownloadFailed indicates a non-success HTTP response when downloading gcloud.
	errGcloudDownloadFailed = errors.New("download failed")
)

const (
	// gcloudDownloadBaseURL is the base URL for downloading Google Cloud SDK archives.
	gcloudDownloadBaseURL = "https://storage.googleapis.com"
	// gcsBucketName is the Google Cloud Storage bucket that holds gcloud SDK objects.
	gcsBucketName = "cloud-sdk-release"
	// gcsAPIURL is the base URL for the Google Cloud Storage JSON API.
	gcsAPIURL = gcloudDownloadBaseURL + "/storage/v1/b/%s/o"
	// gcsDownloadPathTemplate is the path template (without scheme/host) used to
	// download gcloud SDK archives.
	gcsDownloadPathTemplate = "/storage/v1/b/%s/o/%s?alt=media"
	// gcsObjectPrefix is the object path prefix for gcloud SDK downloads.
	gcsObjectPrefix = "google-cloud-sdk"
)

type (
	// GcloudPlugin implements the asdf.Plugin interface for Google Cloud SDK.
	GcloudPlugin struct {
		apiURL string
	}

	// gcsResponse represents the GCS API response.
	gcsResponse struct {
		NextPageToken string      `json:"next_page_token"`
		Items         []gcsObject `json:"items"`
	}

	// gcsObject represents a single object entry from the GCS JSON API.
	gcsObject struct {
		Name string `json:"name"`
	}
)

func (resp *gcsResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if rawItems, ok := raw["items"]; ok {
		var items []gcsObject
		if err := json.Unmarshal(rawItems, &items); err != nil {
			return err
		}

		resp.Items = items
	}

	if rawToken, ok := raw["next_page_token"]; ok {
		var token string
		if err := json.Unmarshal(rawToken, &token); err != nil {
			return err
		}

		resp.NextPageToken = token

		return nil
	}

	if rawToken, ok := raw["nextPageToken"]; ok {
		var token string
		if err := json.Unmarshal(rawToken, &token); err != nil {
			return err
		}

		resp.NextPageToken = token
	}

	return nil
}

// NewGcloudPlugin creates a new gcloud plugin instance.
func NewGcloudPlugin() asdf.Plugin {
	return &GcloudPlugin{
		apiURL: fmt.Sprintf(gcsAPIURL, gcsBucketName),
	}
}

// Name returns the plugin name.
func (*GcloudPlugin) Name() string {
	return "gcloud"
}

// Dependencies returns the list of plugins that must be installed before gcloud.
func (*GcloudPlugin) Dependencies() []string {
	return []string{"python"}
}

// ListBinPaths returns the binary paths for gcloud installations.
func (*GcloudPlugin) ListBinPaths() string {
	return "google-cloud-sdk/bin"
}

// ExecEnv returns environment variables for gcloud execution.
func (*GcloudPlugin) ExecEnv(installPath string) map[string]string {
	return map[string]string{
		"CLOUDSDK_ROOT_DIR": filepath.Join(installPath, "google-cloud-sdk"),
	}
}

// ListLegacyFilenames returns legacy version filenames for gcloud.
func (*GcloudPlugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a legacy gcloud version file.
func (*GcloudPlugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes a gcloud installation.
func (*GcloudPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the gcloud plugin.
func (*GcloudPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Google Cloud SDK (gcloud) - Command-line interface for Google Cloud Platform.
This plugin downloads the Google Cloud SDK from Google Cloud Storage.`,
		Deps: `Requires Python 3.8+ to be installed and available in PATH.`,
		Config: `Environment variables:
  CLOUDSDK_CONFIG - Override gcloud gcloudConfig directory
  CLOUDSDK_PYTHON - Override Python interpreter path`,
		Links: `Homepage: https://cloud.google.com/sdk
Documentation: https://cloud.google.com/sdk/docs
Downloads: https://cloud.google.com/sdk/docs/install`,
	}
}

// ListAll lists all available gcloud versions.
func (plugin *GcloudPlugin) ListAll(ctx context.Context) ([]string, error) {
	versions := make(map[string]bool)
	versionRegex := regexp.MustCompile(`google-cloud-sdk-(\d+\.\d+\.\d+)-linux-x86_64\.tar\.gz$`)

	pageToken := ""

	for {
		url := fmt.Sprintf(
			"%s?prefix=%s&fields=items(name),nextPageToken",
			plugin.apiURL,
			gcsObjectPrefix,
		)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching versions: %w", err)
		}

		var gcsResp gcsResponse
		if err := json.NewDecoder(resp.Body).Decode(&gcsResp); err != nil {
			resp.Body.Close()

			return nil, fmt.Errorf("decoding response: %w", err)
		}

		resp.Body.Close()

		for _, item := range gcsResp.Items {
			matches := versionRegex.FindStringSubmatch(item.Name)
			if len(matches) == 2 {
				versions[matches[1]] = true
			}
		}

		if gcsResp.NextPageToken == "" {
			break
		}

		pageToken = gcsResp.NextPageToken
	}

	result := make([]string, 0, len(versions))
	for v := range versions {
		result = append(result, v)
	}

	sort.Slice(result, func(i, j int) bool {
		return asdf.CompareVersions(result[i], result[j]) < 0
	})

	stable := asdf.FilterVersions(result, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		return stable, nil
	}

	return result, nil
}

// LatestStable returns the latest stable gcloud version.
func (plugin *GcloudPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	return asdf.LatestStableWithQuery(
		ctx,
		query,
		versions,
		errGcloudNoVersionsFound,
		errGcloudNoVersionsMatching,
	)
}

// getObjectName returns the GCS object name for the specified version.
func (*GcloudPlugin) getObjectName(version string) (string, error) {
	var platform string

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			platform = "linux-x86_64"
		case "arm64":
			platform = "linux-arm"
		default:
			return "", fmt.Errorf("%w: %s", errGcloudUnsupportedArch, runtime.GOARCH)
		}

	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			platform = "darwin-x86_64"
		case "arm64":
			platform = "darwin-arm"
		default:
			return "", fmt.Errorf("%w: %s", errGcloudUnsupportedArch, runtime.GOARCH)
		}

	default:
		return "", fmt.Errorf("%w: %s", errGcloudUnsupportedPlatform, runtime.GOOS)
	}

	return fmt.Sprintf("google-cloud-sdk-%s-%s.tar.gz", version, platform), nil
}

// Download downloads the specified gcloud version.
func (plugin *GcloudPlugin) Download(ctx context.Context, version, downloadPath string) error {
	objectName, err := plugin.getObjectName(version)
	if err != nil {
		return err
	}

	filePath := filepath.Join(downloadPath, objectName)
	if info, err := os.Stat(filePath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for gcloud %s", version)

		return nil
	}

	encodedName := strings.ReplaceAll(objectName, "/", "%2F")
	url := gcloudDownloadBaseURL + fmt.Sprintf(gcsDownloadPathTemplate, gcsBucketName, encodedName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading gcloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status: %d", errGcloudDownloadFailed, resp.StatusCode)
	}

	outFile, err := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		asdf.CommonFilePermission,
	)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// Install installs gcloud from the downloaded archive.
func (plugin *GcloudPlugin) Install(
	ctx context.Context,
	version, downloadPath, installPath string,
) error {
	if err := plugin.Download(ctx, version, downloadPath); err != nil {
		return err
	}

	objectName, err := plugin.getObjectName(version)
	if err != nil {
		return err
	}

	archivePath := filepath.Join(downloadPath, objectName)
	if err := asdf.ExtractTarGz(archivePath, installPath); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	return nil
}
