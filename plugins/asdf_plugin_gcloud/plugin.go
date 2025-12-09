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

package asdf_plugin_gcloud

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
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_python"
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

	// ensureToolchains is the function used to ensure toolchains are installed.
	// It is a variable to allow tests to replace it with a fast stub so that
	// Install behavior can be tested without performing real installs.
	ensureToolchains = asdf.EnsureToolchains //nolint:gochecknoglobals // configurable in tests

	// gcloudExtractTarGz is the function used to extract tar.gz archives.
	// It is a variable to allow tests to replace it with a mock implementation
	// that avoids real extraction work while still exercising Install logic.
	gcloudExtractTarGz = asdf.ExtractTarGz //nolint:gochecknoglobals // configurable in tests

	// gcloudHTTPClient is the HTTP client used for API and download requests.
	// It is a variable so tests can replace it with a mock client to avoid
	// real network usage while still exercising non-cached Download and
	// ListAll logic.
	gcloudHTTPClient = http.DefaultClient //nolint:gochecknoglobals // configurable in tests

	// gcloudDownloadBaseURL is the base URL used to construct gcloud download
	// URLs. It is a variable (not a constant) to allow tests to override it
	// with a local httptest.Server, ensuring Download can run fully offline.
	gcloudDownloadBaseURL = "https://storage.googleapis.com" //nolint:gochecknoglobals // configurable in tests

	// installPythonToolchain installs the Python toolchain into an asdf-style
	// installs tree under ASDF_DATA_DIR or ~/.asdf using the shared Python
	// plugin helper. It is a variable so that tests can replace it with a fast
	// stub to avoid performing real Python installs.
	installPythonToolchain = asdf_plugin_python.InstallPythonToolchain //nolint:gochecknoglobals // configurable in tests
)

const (
	// gcsBucketName is the Google Cloud Storage bucket that holds gcloud SDK objects.
	gcsBucketName = "cloud-sdk-release"
	// gcsAPIURL is the base URL for the Google Cloud Storage JSON API.
	gcsAPIURL = "https://storage.googleapis.com/storage/v1/b/%s/o"
	// gcsDownloadPathTemplate is the path template (without scheme/host) used to
	// download gcloud SDK archives.
	gcsDownloadPathTemplate = "/storage/v1/b/%s/o/%s?alt=media"
	// gcsObjectPrefix is the object path prefix for gcloud SDK downloads.
	gcsObjectPrefix = "google-cloud-sdk"
)

type (
	// Plugin implements the asdf.Plugin interface for Google Cloud SDK.
	Plugin struct {
		apiURL  string
		runtime gcloudRuntime
	}

	// gcloudRuntime captures minimal OS/arch runtime information for gcloud.
	gcloudRuntime struct {
		goos string
		arch string
	}

	// gcsResponse represents the GCS API response.
	gcsResponse struct {
		NextPageToken string      `json:"nextPageToken"`
		Items         []gcsObject `json:"items"`
	}

	// gcsObject represents a single object entry from the GCS JSON API.
	gcsObject struct {
		Name string `json:"name"`
	}
)

// defaultGcloudRuntime returns a gcloudRuntime based on the current Go runtime.
func defaultGcloudRuntime() gcloudRuntime {
	return gcloudRuntime{
		goos: runtime.GOOS,
		arch: runtime.GOARCH,
	}
}

// New creates a new gcloud plugin instance.
func New() *Plugin {
	return &Plugin{
		apiURL:  fmt.Sprintf(gcsAPIURL, gcsBucketName),
		runtime: defaultGcloudRuntime(),
	}
}

// NewWithURL creates a new gcloud plugin with custom API URL (for testing).
func NewWithURL(apiURL string) *Plugin {
	return &Plugin{
		apiURL:  apiURL,
		runtime: defaultGcloudRuntime(),
	}
}

// newPluginWithRuntime constructs a Plugin with the provided runtime, used in tests.
func newPluginWithRuntime(apiURL string, rt gcloudRuntime) *Plugin {
	return &Plugin{
		apiURL:  apiURL,
		runtime: rt,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "gcloud"
}

// ListBinPaths returns the binary paths for gcloud installations.
func (*Plugin) ListBinPaths() string {
	return "google-cloud-sdk/bin"
}

// ExecEnv returns environment variables for gcloud execution.
func (*Plugin) ExecEnv(installPath string) map[string]string {
	return map[string]string{
		"CLOUDSDK_ROOT_DIR": filepath.Join(installPath, "google-cloud-sdk"),
	}
}

// ListLegacyFilenames returns legacy version filenames for gcloud.
func (*Plugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a legacy gcloud version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes a gcloud installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the gcloud plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `Google Cloud SDK (gcloud) - Command-line interface for Google Cloud Platform.
This plugin downloads the Google Cloud SDK from Google Cloud Storage.`,
		Deps: `Requires Python 3.8+ to be installed and available in PATH.`,
		Config: `Environment variables:
  CLOUDSDK_CONFIG - Override gcloud config directory
  CLOUDSDK_PYTHON - Override Python interpreter path`,
		Links: `Homepage: https://cloud.google.com/sdk
Documentation: https://cloud.google.com/sdk/docs
Downloads: https://cloud.google.com/sdk/docs/install`,
	}
}

// ListAll lists all available gcloud versions.
func (plugin *Plugin) ListAll(ctx context.Context) ([]string, error) {
	versions := make(map[string]bool)
	versionRegex := regexp.MustCompile(`google-cloud-sdk-(\d+\.\d+\.\d+)-linux-x86_64\.tar\.gz$`)

	pageToken := ""
	for {
		url := fmt.Sprintf("%s?prefix=%s&fields=items(name),nextPageToken", plugin.apiURL, gcsObjectPrefix)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := gcloudHTTPClient.Do(req)
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

	return result, nil
}

// LatestStable returns the latest stable gcloud version.
func (plugin *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errGcloudNoVersionsFound
	}

	if query != "" {
		var filtered []string
		for _, v := range versions {
			if strings.HasPrefix(v, query) {
				filtered = append(filtered, v)
			}
		}

		versions = filtered
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("%w: %s", errGcloudNoVersionsMatching, query)
	}

	return versions[len(versions)-1], nil
}

// getObjectName returns the GCS object name for the specified version.
func (plugin *Plugin) getObjectName(version string) (string, error) {
	goos := plugin.runtime.goos
	arch := plugin.runtime.arch

	var platform string
	switch goos {
	case "linux":
		switch arch {
		case "amd64":
			platform = "linux-x86_64"
		case "arm64":
			platform = "linux-arm"
		default:
			return "", fmt.Errorf("%w: %s", errGcloudUnsupportedArch, arch)
		}

	case "darwin":
		switch arch {
		case "amd64":
			platform = "darwin-x86_64"
		case "arm64":
			platform = "darwin-arm"
		default:
			return "", fmt.Errorf("%w: %s", errGcloudUnsupportedArch, arch)
		}

	default:
		return "", fmt.Errorf("%w: %s", errGcloudUnsupportedPlatform, goos)
	}

	return fmt.Sprintf("google-cloud-sdk-%s-%s.tar.gz", version, platform), nil
}

// Download downloads the specified gcloud version.
func (plugin *Plugin) Download(ctx context.Context, version, downloadPath string) error {
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

	resp, err := gcloudHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading gcloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status: %d", errGcloudDownloadFailed, resp.StatusCode)
	}

	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, asdf.CommonFilePermission)
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
func (plugin *Plugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	if err := ensureToolchains(ctx, "python"); err != nil {
		return err
	}

	if err := installPythonToolchain(ctx); err != nil {
		return err
	}

	objectName, err := plugin.getObjectName(version)
	if err != nil {
		return err
	}

	archivePath := filepath.Join(downloadPath, objectName)

	if err := plugin.extractTarGz(archivePath, installPath); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	return nil
}

// extractTarGz extracts a tar.gz file to the destination directory.
func (*Plugin) extractTarGz(archivePath, destPath string) error {
	return gcloudExtractTarGz(archivePath, destPath)
}
