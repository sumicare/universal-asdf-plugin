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
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPlatform(t *testing.T) {
	t.Parallel()

	platform, err := GetPlatform()
	require.NoError(t, err)
	require.Contains(t, []string{"linux", "darwin", "windows", "freebsd"}, platform)
}

func TestGetArch(t *testing.T) {
	// Not parallel because it sets environment variables
	t.Run("returns the current architecture", func(t *testing.T) {
		arch, err := GetArch()
		require.NoError(t, err)
		require.NotEmpty(t, arch)
	})

	tests := []struct {
		name     string
		env      string
		expected string
		wantErr  bool
	}{
		{"amd64", "amd64", "amd64", false},
		{"x86_64", "x86_64", "amd64", false},
		{"386", "386", "386", false},
		{"i386", "i386", "386", false},
		{"i686", "i686", "386", false},
		{"arm", "arm", "armv6l", false},
		{"arm64", "arm64", "arm64", false},
		{"aarch64", "aarch64", "arm64", false},
		{"ppc64le", "ppc64le", "ppc64le", false},
		{"loong64", "loong64", "loong64", false},
		{"loongarch64", "loongarch64", "loong64", false},
		{"riscv64", "riscv64", "riscv64", false},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run("maps "+tt.name, func(t *testing.T) {
			t.Setenv("ASDF_OVERWRITE_ARCH", tt.env)

			arch, err := GetArch()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, arch)
			}
		})
	}

	t.Run("returns error for unsupported arch", func(t *testing.T) {
		if runtime.GOARCH == "unsupported" {
			t.Skip("cannot test unsupported arch on this platform")
		}

		t.Setenv("ASDF_OVERWRITE_ARCH", "unsupported")

		_, err := GetArch()
		require.Error(t, err)
	})
}

func TestHTTPClient(t *testing.T) {
	t.Parallel()

	client := HTTPClient()
	require.NotNil(t, client)
	require.NotZero(t, client.Timeout)
}

func TestVerifySHA256(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup     func(*testing.T) (string, string)
		name      string
		errSubstr string
		wantErr   bool
	}{
		{
			name: "verifies correct checksum",
			setup: func(*testing.T) (string, string) {
				path := filepath.Join(t.TempDir(), "test.txt")
				require.NoError(t, os.WriteFile(path, []byte("test content"), CommonFilePermission))

				return path, "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
			},
			wantErr: false,
		},
		{
			name: "returns error for incorrect checksum",
			setup: func(*testing.T) (string, string) {
				path := filepath.Join(t.TempDir(), "test.txt")
				require.NoError(t, os.WriteFile(path, []byte("test content"), CommonFilePermission))

				return path, "wronghash"
			},
			wantErr:   true,
			errSubstr: "checksum mismatch",
		},
		{
			name: "returns error for nonexistent file",
			setup: func(*testing.T) (string, string) {
				return "/nonexistent/file", "somehash"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path, hash := tt.setup(t)

			err := VerifySHA256(path, hash)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errSubstr != "" {
					require.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	t.Parallel()

	t.Run("creates nested directories", func(t *testing.T) {
		t.Parallel()

		nestedPath := filepath.Join(t.TempDir(), "a", "b", "c")
		require.NoError(t, EnsureDir(nestedPath))

		info, err := os.Stat(nestedPath)
		require.NoError(t, err)
		require.True(t, info.IsDir())
	})

	t.Run("succeeds if directory already exists", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, EnsureDir(t.TempDir()))
	})
}

func TestFilterVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		versions  []string
		predicate func(string) bool
		expected  []string
	}{
		{
			name:     "filters based on predicate",
			versions: []string{"1.0.0", "1.1.0", "2.0.0", "2.1.0"},
			predicate: func(v string) bool {
				return v[0] == '1'
			},
			expected: []string{"1.0.0", "1.1.0"},
		},
		{
			name:     "returns empty slice if no matches",
			versions: []string{"1.0.0", "1.1.0"},
			predicate: func(v string) bool {
				return v[0] == '3'
			},
			expected: make([]string, 0),
		},
		{
			name:     "filters by predicate with prefix",
			versions: []string{"1.20.0", "1.21.0", "1.21.5", "2.0.0"},
			predicate: func(v string) bool {
				return strings.HasPrefix(v, "1.21")
			},
			expected: []string{"1.21.0", "1.21.5"},
		},
		{
			name:     "returns all versions when predicate always returns true",
			versions: []string{"1.20.0", "1.21.0"},
			predicate: func(_ string) bool {
				return true
			},
			expected: []string{"1.20.0", "1.21.0"},
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filtered := FilterVersions(tt.versions, tt.predicate)
			require.Equal(t, tt.expected, filtered)

			if len(tt.expected) == 0 {
				require.Empty(t, filtered)
			}
		})
	}
}

func TestSortVersions(t *testing.T) {
	t.Parallel()

	versions := []string{"2.0.0", "1.0.0", "1.1.0", "10.0.0"}
	SortVersions(versions)
	require.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0", "10.0.0"}, versions)
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		a            string
		b            string
		expectedSign int
	}{
		{"1.0.0 < 2.0.0", "1.0.0", "2.0.0", -1},
		{"2.0.0 > 1.0.0", "2.0.0", "1.0.0", 1},
		{"1.0.0 == 1.0.0", "1.0.0", "1.0.0", 0},
		{"1.9 < 1.10", "1.9", "1.10", -1},
		{"1.21.0 > 1.20.0", "1.21.0", "1.20.0", 1},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CompareVersions(tt.a, tt.b)
			switch {
			case tt.expectedSign < 0:
				require.Less(t, result, 0)
			case tt.expectedSign > 0:
				require.Greater(t, result, 0)
			default:
				require.Equal(t, 0, result)
			}
		})
	}
}

func TestParseVersionParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{"extracts numeric parts", "1.21.0", []int{1, 21, 0}},
		{"handles versions with prefixes", "go1.21.0", []int{1, 21, 0}},
		{"handles rc versions", "1.21rc1", []int{1, 21, 1}},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parts := ParseVersionParts(tt.version)
			require.Equal(t, tt.expected, parts)
		})
	}
}

func TestReadLegacyVersionFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*testing.T) string
		expected string
		wantErr  bool
	}{
		{
			name: "reads and trims version file",
			setup: func(t *testing.T) string {
				t.Helper()

				path := filepath.Join(t.TempDir(), ".version")
				require.NoError(t, os.WriteFile(path, []byte("  1.21.0  \n"), CommonFilePermission))

				return path
			},
			expected: "1.21.0",
		},
		{
			name:    "returns error for nonexistent file",
			setup:   func(*testing.T) string { return "/nonexistent/file" },
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.setup(t)

			version, err := ReadLegacyVersionFile(path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, version)
			}
		})
	}
}

func TestMsgAndErr(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() { Msgf("test %s", "message") })
	require.NotPanics(t, func() { Errf("test %s", "error") })
}

func TestDownloadFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test.txt" {
			_, err := w.Write([]byte("test content"))
			require.NoError(t, err)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		setup     func(*testing.T) string
		name      string
		url       string
		wantErr   bool
		checkFile bool
	}{
		{
			name: "downloads file successfully",
			url:  server.URL + "/test.txt",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "downloaded.txt")
			},
			checkFile: true,
		},
		{
			name: "returns error for 404",
			url:  server.URL + "/notfound",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "notfound.txt")
			},
			wantErr: true,
		},
		{
			name: "returns error for invalid URL",
			url:  "http://invalid.invalid.invalid:99999/file",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "invalid.txt")
			},
			wantErr: true,
		},
		{
			name: "returns error when cannot create destination file",
			url:  server.URL + "/test.txt",
			setup: func(t *testing.T) string {
				t.Helper()
				return "/nonexistent/path/file.txt"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			destPath := tt.setup(t)

			err := DownloadFile(t.Context(), tt.url, destPath)
			if !tt.wantErr {
				require.NoError(t, err)

				if tt.checkFile {
					content, err := os.ReadFile(destPath)
					require.NoError(t, err)
					require.Equal(t, "test content", string(content))
				}

				return
			}

			require.Error(t, err)
		})
	}
}

func TestDownloadString(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/content" {
			_, err := w.Write([]byte("string content"))
			require.NoError(t, err)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "downloads string successfully",
			url:      server.URL + "/content",
			expected: "string content",
		},
		{
			name:    "returns error for 404",
			url:     server.URL + "/notfound",
			wantErr: true,
		},
		{
			name:    "returns error for invalid URL",
			url:     "http://invalid.invalid.invalid:99999/content",
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, err := DownloadString(t.Context(), tt.url)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, content)
			}
		})
	}
}

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		expected string
		versions []string
	}{
		{
			name:     "returns latest version",
			versions: []string{"1.0.0", "2.0.0", "1.5.0"},
			expected: "2.0.0",
		},
		{
			name:     "returns latest matching pattern",
			versions: []string{"1.0.0", "2.0.0", "1.5.0"},
			pattern:  "1",
			expected: "1.5.0",
		},
		{
			name:     "prefers stable versions over prereleases",
			versions: []string{"1.0.0", "1.1.0-rc1", "1.1.0"},
			expected: "1.1.0",
		},
		{
			name:     "falls back to prereleases when no stable versions exist",
			versions: []string{"1.1.0-rc1", "1.1.0-beta1"},
			expected: "1.1.0-beta1",
		},
		{
			name:     "returns empty string if no match",
			versions: []string{"1.0.0", "2.0.0"},
			pattern:  "3",
			expected: "",
		},
		{
			name:     "returns empty string for empty list",
			versions: make([]string, 0),
			expected: "",
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			latest := LatestVersion(tt.versions, tt.pattern)
			require.Equal(t, tt.expected, latest)
		})
	}
}

var (
	errTestNoVersions = errors.New("no versions found")
	errTestNoMatching = errors.New("no matching versions")
)

func TestLatestStableWithQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr   error
		name      string
		query     string
		expected  string
		errSubstr string
		versions  []string
	}{
		{
			name:     "returns latest stable version without query",
			versions: []string{"1.0.0", "1.1.0", "2.0.0-beta", "2.0.0"},
			expected: "2.0.0",
		},
		{
			name:     "filters by query prefix",
			query:    "1.",
			versions: []string{"1.0.0", "1.1.0", "2.0.0", "2.1.0"},
			expected: "1.1.0",
		},
		{
			name:     "filters out prerelease versions",
			versions: []string{"1.0.0", "1.1.0-alpha", "1.1.0-beta", "1.2.0-rc1"},
			expected: "1.0.0",
		},
		{
			name:     "returns latest prerelease if no stable versions exist",
			versions: []string{"1.0.0-alpha", "1.0.0-beta", "1.1.0-rc1"},
			expected: "1.1.0-rc1",
		},
		{
			name:    "returns error when no versions provided",
			wantErr: errTestNoVersions,
		},
		{
			name:      "returns error when query matches no versions",
			query:     "3.",
			versions:  []string{"1.0.0", "1.1.0", "2.0.0"},
			wantErr:   errTestNoMatching,
			errSubstr: "3.",
		},
		{
			name:     "handles complex version patterns",
			query:    "1.2",
			versions: []string{"1.2.3", "1.2.4-alpha", "1.2.4", "1.3.0-beta", "1.3.0"},
			expected: "1.2.4",
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			latest, err := LatestStableWithQuery(t.Context(), tt.query, tt.versions, errTestNoVersions, errTestNoMatching)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				if tt.errSubstr != "" {
					require.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, latest)
			}
		})
	}
}
