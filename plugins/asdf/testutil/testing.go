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

package testutil

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// TestingT is an interface that both *testing.T and ginkgo.GinkgoT satisfy.
type TestingT interface {
	// Helper marks the calling function as a testing helper function.
	Helper()
	// Fatal is equivalent to log.Fatal in the standard library.
	Fatal(args ...any)
	// Fatalf is equivalent to log.Fatalf in the standard library.
	Fatalf(format string, args ...any)
}

// TestDataDir returns the .tmp directory path relative to the caller's package.
// This is used for test installs to avoid polluting the system.
func TestDataDir(tb TestingT) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		tb.Fatal("failed to get caller information")
	}

	tmpDir := filepath.Join(filepath.Dir(file), ".tmp")
	if err := os.MkdirAll(tmpDir, CommonDirectoryPermission); err != nil {
		tb.Fatalf("failed to create test data dir: %v", err)
	}

	return tmpDir
}

// SetupTestEnv sets ASDF_DATA_DIR to the .tmp directory for testing.
// Returns a cleanup function to restore the original value.
func SetupTestEnv(tb TestingT) func() {
	tb.Helper()

	tmpDir := TestDataDir(tb)
	original := os.Getenv("ASDF_DATA_DIR")

	os.Setenv("ASDF_DATA_DIR", tmpDir)

	return func() {
		if original == "" {
			os.Unsetenv("ASDF_DATA_DIR")
		} else {
			os.Setenv("ASDF_DATA_DIR", original)
		}
	}
}

// TestInstallDir returns a subdirectory within .tmp for a specific test install.
func TestInstallDir(tb TestingT, name string) string {
	tb.Helper()

	dir := filepath.Join(TestDataDir(tb), "installs", name)
	if err := os.MkdirAll(dir, CommonDirectoryPermission); err != nil {
		tb.Fatalf("failed to create install dir: %v", err)
	}

	return dir
}

// TestDownloadDir returns a subdirectory within .tmp for test downloads.
func TestDownloadDir(tb TestingT, name string) string {
	tb.Helper()

	dir := filepath.Join(TestDataDir(tb), "downloads", name)
	if err := os.MkdirAll(dir, CommonDirectoryPermission); err != nil {
		tb.Fatalf("failed to create download dir: %v", err)
	}

	return dir
}

// TestBuildDir returns a subdirectory within .tmp for build tools (node-build, python-build).
func TestBuildDir(tb TestingT, name string) string {
	tb.Helper()

	dir := filepath.Join(TestDataDir(tb), "build-tools", name)
	if err := os.MkdirAll(dir, CommonDirectoryPermission); err != nil {
		tb.Fatalf("failed to create build dir: %v", err)
	}

	return dir
}

// CreateTestFile creates a file with the given content in a temp directory.
// Returns the file path and a cleanup function.
func CreateTestFile(tb TestingT, name string, content []byte) (string, func()) {
	tb.Helper()

	tmpDir, err := os.MkdirTemp("", "asdf-test-*")
	if err != nil {
		tb.Fatalf("failed to create temp dir: %v", err)
	}

	filePath := filepath.Join(tmpDir, name)
	if err := os.WriteFile(filePath, content, CommonFilePermission); err != nil {
		os.RemoveAll(tmpDir)
		tb.Fatalf("failed to write test file: %v", err)
	}

	return filePath, func() { os.RemoveAll(tmpDir) }
}

// CreateTestDir creates a temp directory with optional subdirectories.
// Returns the directory path and a cleanup function.
func CreateTestDir(tb TestingT, subdirs ...string) (string, func()) {
	tb.Helper()

	tmpDir, err := os.MkdirTemp("", "asdf-test-*")
	if err != nil {
		tb.Fatalf("failed to create temp dir: %v", err)
	}

	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), CommonDirectoryPermission); err != nil {
			os.RemoveAll(tmpDir)
			tb.Fatalf("failed to create subdir %s: %v", subdir, err)
		}
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

// GoldieTestDataPath returns the path to the testdata directory for a plugin.
// The path is relative to the caller's package directory.
func GoldieTestDataPath(tb TestingT) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		tb.Fatal("failed to get caller information")
	}

	return filepath.Join(filepath.Dir(file), "testdata")
}

// ReadGoldieVersions reads version list from a goldie file (e.g., "argo_list_all.golden").
// Returns the versions as a slice of strings.
func ReadGoldieVersions(testdataPath, goldenFile string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(testdataPath, goldenFile))
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, nil
	}

	return strings.Split(content, "\n"), nil
}

// ReadGoldieLatest reads the latest version from a goldie file (e.g., "argo_latest_stable.golden").
func ReadGoldieLatest(testdataPath, goldenFile string) (string, error) {
	data, err := os.ReadFile(filepath.Join(testdataPath, goldenFile))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// MaximizeVersion converts a version string to its "maximum" form by replacing
// each digit sequence with 9s of the same length.
// Examples:
//   - "3.7.1" -> "9.9.9"
//   - "24.9.1" -> "99.9.9"
//   - "v3.7.1" -> "v9.9.9"
//   - "stable-24" -> "stable-99"
func MaximizeVersion(version string) string {
	re := regexp.MustCompile(`\d+`)

	return re.ReplaceAllStringFunc(version, func(match string) string {
		return strings.Repeat("9", len(match))
	})
}

// VersionsToTags converts version strings to tag format by adding "v" prefix if needed.
//
//nolint:revive // we're fine with this flag
func VersionsToTags(versions []string, addVPrefix bool) []string {
	tags := make([]string, 0, len(versions))
	for _, v := range versions {
		if addVPrefix && !strings.HasPrefix(v, "v") {
			tags = append(tags, "v"+v)
		} else {
			tags = append(tags, v)
		}
	}

	return tags
}

// GoldieFileExists checks if a goldie file exists in the testdata directory.
func GoldieFileExists(testdataPath, goldenFile string) bool {
	_, err := os.Stat(filepath.Join(testdataPath, goldenFile))
	return err == nil
}

// GenerateFilterPattern creates a pattern that matches a subset of versions.
// It takes the versions list and creates a pattern that would match older versions.
// For example, if versions are ["3.6.0", "3.7.0", "3.7.1"], it returns "3.6" to match 3.6.x.
func GenerateFilterPattern(versions []string) string {
	if len(versions) < 2 {
		return ""
	}

	first := versions[0]

	parts := strings.Split(first, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}

	return first
}
