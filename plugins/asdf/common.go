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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type (
	// Plugin defines the interface that all asdf plugins must implement.
	// Based on https://asdf-vm.com/plugins/create.html
	Plugin interface {
		// Name returns the plugin name (e.g., "golang", "python", "nodejs").
		Name() string

		// ListAll returns all available versions for this tool.
		ListAll(ctx context.Context) ([]string, error)

		// Download downloads the specified version to downloadPath.
		Download(ctx context.Context, version, downloadPath string) error

		// Install installs the specified version from downloadPath to installPath.
		Install(ctx context.Context, version, downloadPath, installPath string) error

		// ListBinPaths returns the relative paths to directories containing binaries.
		ListBinPaths() string

		// ExecEnv returns environment variables to set when executing tool binaries.
		ExecEnv(installPath string) map[string]string

		// Uninstall removes the specified version.
		Uninstall(ctx context.Context, installPath string) error

		// LatestStable returns the latest stable version matching the query.
		LatestStable(ctx context.Context, query string) (string, error)

		// ListLegacyFilenames returns filenames to check for legacy version files.
		ListLegacyFilenames() []string

		// ParseLegacyFile parses a legacy version file and returns the version.
		ParseLegacyFile(path string) (string, error)

		// Help returns help information for the plugin.
		Help() PluginHelp
	}

	// PluginHelp contains help information for a plugin.
	PluginHelp struct {
		// Overview is a general description of the plugin and tool.
		Overview string
		// Deps lists system dependencies required by the plugin.
		Deps string
		// Config describes configuration options and environment variables.
		Config string
		// Links provides useful links for the tool.
		Links string
	}

	// InstallConfig holds configuration for installation.
	InstallConfig struct {
		Version      string
		InstallType  string
		DownloadPath string
		InstallPath  string
	}
)

var (
	// errPlatformNotSupported is returned when the running OS cannot be mapped to a supported platform.
	errPlatformNotSupported = errors.New("platform not supported")
	// errArchNotSupported is returned when the running architecture cannot be mapped to a supported arch.
	errArchNotSupported = errors.New("arch not supported")
	// errDownloadFailed indicates that an HTTP download completed with a non-success status code.
	errDownloadFailed = errors.New("download failed")
	// errChecksumMismatchGeneric is returned when a computed checksum does not match the expected value.
	errChecksumMismatchGeneric = errors.New("checksum mismatch")
	// errInvalidArchiveFilePathTar is returned when a tar entry would escape the extraction directory.
	errInvalidArchiveFilePathTar = errors.New("invalid file path in tar archive")
	// errInvalidArchiveFilePathZip is returned when a zip entry would escape the extraction directory.
	errInvalidArchiveFilePathZip = errors.New("invalid file path in zip archive")
)

const (
	// CommonFilePermission is the default file permission used when creating files.
	CommonFilePermission os.FileMode = 0o600
	// CommonDirectoryPermission is the default permission used when creating directories.
	CommonDirectoryPermission os.FileMode = 0o755
	// CommonExecutablePermission is the default permission used when creating directories.
	CommonExecutablePermission os.FileMode = 0o755
	// ExecutablePermissionMask is the mask used to set executable permissions.
	ExecutablePermissionMask os.FileMode = 0o111
)

// GetPlatform returns the current platform (linux, darwin, freebsd).
func GetPlatform() (string, error) {
	platform := strings.ToLower(runtime.GOOS)
	switch platform {
	case "linux", "darwin", "freebsd":
		return platform, nil
	default:
		return "", fmt.Errorf("%w: %s", errPlatformNotSupported, platform)
	}
}

// GetArch returns the current architecture in Go download format.
func GetArch() (string, error) {
	archOverride := os.Getenv("ASDF_OVERWRITE_ARCH")

	arch := runtime.GOARCH
	if archOverride != "" {
		arch = archOverride
	}

	switch arch {
	case "amd64", "x86_64":
		return "amd64", nil
	case "386", "i386", "i686":
		return "386", nil
	case "arm":
		return "armv6l", nil
	case "arm64", "aarch64":
		return "arm64", nil
	case "ppc64le":
		return "ppc64le", nil
	case "loong64", "loongarch64":
		return "loong64", nil
	case "riscv64":
		return "riscv64", nil
	default:
		return "", fmt.Errorf("%w: %s", errArchNotSupported, arch)
	}
}

// HTTPClient returns an HTTP client with reasonable defaults.
func HTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Minute,
	}
}

// DownloadFile downloads a file from URL to the specified path.
func DownloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := HTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status %d for %s", errDownloadFailed, resp.StatusCode, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", destPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("writing file %s: %w", destPath, err)
	}

	return nil
}

// DownloadString downloads content from URL and returns it as a string.
func DownloadString(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w with status %d for %s", errDownloadFailed, resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(body), nil
}

// VerifySHA256 verifies the SHA256 checksum of a file.
func VerifySHA256(filePath, expectedHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}

	actualHash := hex.EncodeToString(h.Sum(nil))

	trimmedExpectedHash := strings.TrimSpace(strings.Split(expectedHash, " ")[0])

	if actualHash != trimmedExpectedHash {
		return fmt.Errorf("%w: expected %s, got %s", errChecksumMismatchGeneric, trimmedExpectedHash, actualHash)
	}

	return nil
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, CommonDirectoryPermission)
}

// Msgf prints a success message to stderr with formatting.
func Msgf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[32m"+format+"\033[39m\n", args...)
}

// Errf prints an error message to stderr with formatting.
func Errf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[31m"+format+"\033[39m\n", args...)
}

// SortVersions sorts version strings in semver order.
func SortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return CompareVersions(versions[i], versions[j]) < 0
	})
}

// CompareVersions compares two version strings.
// Returns negative if a < b, positive if a > b, zero if equal.
func CompareVersions(a, b string) int {
	partsA := ParseVersionParts(a)
	partsB := ParseVersionParts(b)

	maxLen := max(len(partsA), len(partsB))
	for i := range maxLen {
		var partA, partB int
		if i < len(partsA) {
			partA = partsA[i]
		}

		if i < len(partsB) {
			partB = partsB[i]
		}

		if partA != partB {
			return partA - partB
		}
	}

	return 0
}

// ParseVersionParts extracts numeric parts from a version string.
func ParseVersionParts(version string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(version, -1)

	parts := make([]int, 0, len(matches))
	for i := range matches {
		r, _ := strconv.Atoi(matches[i]) //nolint:errcheck // it should be fine

		parts = append(parts, r)
	}

	return parts
}

// FilterVersions filters versions based on a predicate function.
func FilterVersions(versions []string, predicate func(string) bool) []string {
	result := make([]string, 0, len(versions))
	for _, v := range versions {
		if predicate(v) {
			result = append(result, v)
		}
	}

	return result
}

// ReadLegacyVersionFile reads and parses a legacy version file.
func ReadLegacyVersionFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// IsOnline returns true if ONLINE env var is set, enabling network tests.
func IsOnline() bool {
	v := os.Getenv("ONLINE")
	return v == "1" || strings.EqualFold(v, "true")
}

// LatestVersion returns the latest version from a list, optionally filtered by pattern.
func LatestVersion(versions []string, pattern string) string {
	filtered := versions
	if pattern != "" {
		filtered = FilterVersions(versions, func(v string) bool {
			return strings.HasPrefix(v, pattern)
		})
	}

	if len(filtered) == 0 {
		return ""
	}

	SortVersions(filtered)

	return filtered[len(filtered)-1]
}
