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

package asdf_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
	githubmock "github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

var (
	errTestDownloadFailed    = errors.New("download failed")
	errTestPreBuildFailed    = errors.New("pre-build failed")
	errTestPostInstallFailed = errors.New("post-install failed")
)

// TestSourceBuildPluginListAllUsesReleases verifies ListAll lists stable versions from releases.
func TestSourceBuildPluginListAllUsesReleases(t *testing.T) {
	t.Parallel()

	srv := githubmock.NewServer()
	t.Cleanup(srv.Close)

	srv.AddReleases("o", "r", []string{"v1.2.0", "v1.2.1", "v1.3.0-rc.1"})

	client := github.NewClientWithHTTP(srv.HTTPServer.Client(), srv.URL())

	useTags := false
	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		UseTags:       useTags,
		RepoOwner:     "o",
		RepoName:      "r",
		VersionPrefix: "v",
		Name:          "tool",
	})
	plugin.WithGithubClient(client)

	versions, err := plugin.ListAll(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"1.2.0", "1.2.1"}, versions)
}

// TestSourceBuildPluginListAllUsesTags verifies ListAll honors tag prefix and version filter when listing tags.
func TestSourceBuildPluginListAllUsesTags(t *testing.T) {
	t.Parallel()

	srv := githubmock.NewServer()
	t.Cleanup(srv.Close)

	srv.AddTags("o", "r", []string{"v2.0.0", "v2.1.0", "junk", "v1.0.0"})

	client := github.NewClientWithHTTP(srv.HTTPServer.Client(), srv.URL())

	useTags := true
	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		UseTags:       useTags,
		RepoOwner:     "o",
		RepoName:      "r",
		VersionPrefix: "v",
		VersionFilter: `^2\.`,
		Name:          "tool",
	})
	plugin.WithGithubClient(client)

	versions, err := plugin.ListAll(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"2.0.0", "2.1.0"}, versions)
}

// TestSourceBuildPluginLatestStableErrors verifies LatestStable error behavior when no versions exist.
func TestSourceBuildPluginLatestStableErrors(t *testing.T) {
	t.Parallel()

	useTags := false
	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		UseTags:       useTags,
		RepoOwner:     "o",
		RepoName:      "r",
		VersionPrefix: "v",
		Name:          "tool",
	})

	plugin.WithGithubClient(github.NewClientWithHTTP(&http.Client{}, "http://127.0.0.1:1"))

	_, err := plugin.LatestStable(t.Context(), "")
	require.Error(t, err)
}

func TestSourceBuildPluginName(t *testing.T) {
	t.Parallel()

	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name: "test-plugin",
	})

	require.Equal(t, "test-plugin", plugin.Name())
}

func TestSourceBuildPluginDownload(t *testing.T) {
	t.Parallel()

	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name: "test-plugin",
	})

	err := plugin.Download(t.Context(), "1.0.0", "/tmp/download")
	require.NoError(t, err)
}

func TestSourceBuildPluginListBinPaths(t *testing.T) {
	t.Parallel()

	t.Run("returns configured bin dir", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			Name:   "test-plugin",
			BinDir: "custom/bin",
		})

		require.Equal(t, "custom/bin", plugin.ListBinPaths())
	})

	t.Run("returns default bin when not configured", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			Name: "test-plugin",
		})

		require.Equal(t, "bin", plugin.ListBinPaths())
	})
}

func TestSourceBuildPluginExecEnv(t *testing.T) {
	t.Parallel()

	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name: "test-plugin",
	})

	env := plugin.ExecEnv("/test/install")
	require.Empty(t, env)
}

func TestSourceBuildPluginUninstall(t *testing.T) {
	t.Parallel()

	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name: "test-plugin",
	})

	err := plugin.Uninstall(t.Context(), "/tmp/install")
	require.NoError(t, err)
}

func TestSourceBuildPluginListLegacyFilenames(t *testing.T) {
	t.Parallel()

	t.Run("returns configured legacy filenames", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			Name:            "test-plugin",
			LegacyFilenames: []string{".test-version", "test.config"},
		})

		filenames := plugin.ListLegacyFilenames()
		require.Equal(t, []string{".test-version", "test.config"}, filenames)
	})

	t.Run("returns empty slice when not configured", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			Name: "test-plugin",
		})

		filenames := plugin.ListLegacyFilenames()
		require.Empty(t, filenames)
	})
}

func TestSourceBuildPluginParseLegacyFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	versionFile := filepath.Join(tempDir, ".test-version")
	require.NoError(t, os.WriteFile(versionFile, []byte("1.0.0\n"), 0o600))

	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name: "test-plugin",
	})

	version, err := plugin.ParseLegacyFile(versionFile)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", version)
}

func TestSourceBuildPluginHelp(t *testing.T) {
	t.Parallel()

	plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
		Name: "test-plugin",
	})

	help := plugin.Help()
	require.Empty(t, help.Overview)
	require.Empty(t, help.Deps)
	require.Empty(t, help.Config)
	require.Empty(t, help.Links)
}

func TestRenderSourceBuildTemplate(t *testing.T) {
	t.Parallel()

	cfg := &asdf.SourceBuildPluginConfig{
		RepoOwner:     "owner",
		RepoName:      "repo",
		Name:          "plugin",
		VersionPrefix: "v",
	}

	template := "{{.RepoOwner}}/{{.RepoName}}/{{.Name}}-{{.VersionPrefix}}{{.Version}}"
	result := asdf.RenderSourceBuildTemplateForTests(template, cfg, "1.2.3")
	require.Equal(t, "owner/repo/plugin-v1.2.3", result)
}

func TestSourceBuildPluginInstall(t *testing.T) {
	t.Parallel()

	t.Run("returns error when BuildVersion is nil", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{})
		err := plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.ErrorIs(t, err, asdf.ErrSourceBuildNoBuildStepForTests())
	})

	t.Run("full install flow with tar.gz", func(t *testing.T) {
		t.Parallel()

		// Prepare archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{
			"repo-1.0.0/bin/tool": "binary content",
		})

		archiveContent, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.URL.Path, "archive") {
					writer.WriteHeader(http.StatusOK)

					_, err := writer.Write(archiveContent)
					if err != nil {
						t.Errorf("writing archive response: %v", err)
					}

					return
				}

				http.NotFound(writer, req)
			}),
		)
		defer server.Close()

		buildCalled := false
		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			Name:              "tool",
			RepoName:          "repo",
			ArchiveType:       "tar.gz",
			SourceURLTemplate: server.URL + "/archive/{{.Version}}.tar.gz",
			BuildVersion: func(_ context.Context, _, sourceDir, installPath string) error {
				buildCalled = true
				// Simulate build by copying binary to install path
				return asdf.CopyDir(sourceDir, installPath)
			},
			ExpectedArtifacts: []string{"bin/tool"},
		})

		installPath := t.TempDir()

		err = plugin.Install(t.Context(), "1.0.0", "", installPath)
		require.NoError(t, err)
		require.True(t, buildCalled)

		// Check artifact exists and is executable
		info, err := os.Stat(filepath.Join(installPath, "bin", "tool"))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("skips download if exists", func(t *testing.T) {
		t.Parallel()

		downloadCalled := false

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				downloadCalled = true

				writer.WriteHeader(http.StatusNotFound)
			}),
		)
		defer server.Close()

		tempDir := t.TempDir()
		downloadPath := filepath.Join(tempDir, "downloads")
		require.NoError(t, os.MkdirAll(downloadPath, 0o755))

		// Create a dummy archive
		archivePath := filepath.Join(downloadPath, "repo-1.0.0.tar.gz")
		CreateTestTarGz(t, archivePath, map[string]string{
			"repo-1.0.0/file": "content",
		})

		minSize := int64(0)
		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:          "repo",
			ArchiveType:       "tar.gz",
			SourceURLTemplate: server.URL + "/should-not-call",
			MinArchiveSize:    &minSize,
			BuildVersion: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		})

		installPath := t.TempDir()
		err := plugin.Install(t.Context(), "1.0.0", downloadPath, installPath)
		require.NoError(t, err)
		require.False(t, downloadCalled)
	})

	t.Run("uses SourceURLFunc if provided", func(t *testing.T) {
		t.Parallel()

		// Prepare archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{"repo-1.0.0/f": "c"})

		archiveContent, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/custom/1.0.0.tar.gz" {
					writer.WriteHeader(http.StatusOK)

					_, err := writer.Write(archiveContent)
					if err != nil {
						t.Errorf("writing archive response: %v", err)
					}

					return
				}

				http.NotFound(writer, req)
			}),
		)
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:    "repo",
			ArchiveType: "tar.gz",
			SourceURLFunc: func(_ context.Context, version string) (string, error) {
				return server.URL + "/custom/" + version + ".tar.gz", nil
			},
			BuildVersion: func(_ context.Context, _, _, _ string) error { return nil },
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.NoError(t, err)
	})

	t.Run("extracts zip archive", func(t *testing.T) {
		t.Parallel()

		// Prepare archive
		tempDir := t.TempDir()
		zipPath := filepath.Join(tempDir, "archive.zip")
		CreateTestZip(t, zipPath, map[string]string{"repo-1.0.0/f": "c"})

		archiveContent, err := os.ReadFile(zipPath)
		require.NoError(t, err)

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)

				_, err := writer.Write(archiveContent)
				if err != nil {
					t.Errorf("writing archive response: %v", err)
				}
			}),
		)
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:            "repo",
			ArchiveType:         "zip",
			ArchiveNameTemplate: "repo-{{.Version}}.zip",
			SourceURLTemplate:   server.URL + "/archive.zip",
			BuildVersion: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.NoError(t, err)
	})

	t.Run("auto detects extracted directory", func(t *testing.T) {
		t.Parallel()

		// Prepare archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{
			"unexpected-dir-name/f": "c",
		})

		archiveContent, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)

				_, err := writer.Write(archiveContent)
				if err != nil {
					t.Errorf("writing archive response: %v", err)
				}
			}),
		)
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:               "repo",
			ArchiveType:            "tar.gz",
			AutoDetectExtractedDir: true,
			SourceURLTemplate:      server.URL + "/archive.tar.gz",
			BuildVersion: func(_ context.Context, _, sourceDir, _ string) error {
				// Verify we found the right dir
				if !strings.HasSuffix(sourceDir, "unexpected-dir-name") {
					return os.ErrNotExist
				}

				return nil
			},
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.NoError(t, err)
	})

	t.Run("supports PostInstallVersion", func(t *testing.T) {
		t.Parallel()

		// Prepare archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{"repo-1.0.0/f": "c"})

		archiveContent, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		postInstallCalled := false

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)

				_, err := writer.Write(archiveContent)
				if err != nil {
					t.Errorf("writing archive response: %v", err)
				}
			}),
		)
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:          "repo",
			SourceURLTemplate: server.URL,
			BuildVersion:      func(_ context.Context, _, _, _ string) error { return nil },
			PostInstallVersion: func(_ context.Context, _, _ string) error {
				postInstallCalled = true

				return nil
			},
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.NoError(t, err)
		require.True(t, postInstallCalled)
	})

	t.Run("fast path: skips install if artifacts exist", func(t *testing.T) {
		t.Parallel()

		installPath := t.TempDir()
		binDir := filepath.Join(installPath, "bin")
		require.NoError(t, os.MkdirAll(binDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool"), []byte("bin"), 0o600))
		require.NoError(t, os.Chmod(filepath.Join(binDir, "tool"), 0o755))

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			BuildVersion: func(_ context.Context, _, _, _ string) error {
				return os.ErrExist // Should not be called
			},
			ExpectedArtifacts: []string{"bin/tool"},
		})

		err := plugin.Install(t.Context(), "1.0.0", "", installPath)
		require.NoError(t, err)
	})

	t.Run("returns error when download fails", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName: "repo",
			DownloadFile: func(_ context.Context, _, _ string) error {
				return errTestDownloadFailed
			},
			BuildVersion:      func(_ context.Context, _, _, _ string) error { return nil },
			SourceURLTemplate: "http://example.invalid",
		})

		err := plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "downloading source")
	})

	t.Run("returns error when extract fails", func(t *testing.T) {
		t.Parallel()

		// Create invalid archive
		tempDir := t.TempDir()
		archivePath := filepath.Join(tempDir, "archive.tar.gz")
		require.NoError(t, os.WriteFile(archivePath, []byte("invalid"), 0o600))

		server := httptest.NewServer(
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)

				_, err := writer.Write([]byte("invalid"))
				// We can't use require.NoError here easily as it's a separate goroutine usually,
				// but for httptest it runs in the same process. However, panicking in a handler
				// might be caught by net/http recovery.
				// Best effort: log if error. But since we don't have *testing.T here easily...
				// Actually we are inside t.Run, we can assume it works or ignore.
				// The lint error is errcheck. We can ignore it with blank identifier if we really don't care,
				// or better, check it.
				if err != nil {
					panic(err)
				}
			}),
		)
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:          "repo",
			SourceURLTemplate: server.URL,
			BuildVersion:      func(_ context.Context, _, _, _ string) error { return nil },
		})

		err := plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "extracting tar.gz")
	})

	t.Run("returns error when PreBuildVersion fails", func(t *testing.T) {
		t.Parallel()

		// Prepare valid archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{
			"repo-1.0.0/f": "c",
		})

		content, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(content)
			if err != nil {
				panic(err)
			}
		}))
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:          "repo",
			SourceURLTemplate: server.URL,
			PreBuildVersion: func(_ context.Context, _, _ string) error {
				return errTestPreBuildFailed
			},
			BuildVersion: func(_ context.Context, _, _, _ string) error { return nil },
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "pre-build failed")
	})

	t.Run("returns error when PostInstallVersion fails", func(t *testing.T) {
		t.Parallel()

		// Prepare valid archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{
			"repo-1.0.0/f": "c",
		})

		content, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(content)
			if err != nil {
				panic(err)
			}
		}))
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:          "repo",
			SourceURLTemplate: server.URL,
			BuildVersion:      func(_ context.Context, _, _, _ string) error { return nil },
			PostInstallVersion: func(_ context.Context, _, _ string) error {
				return errTestPostInstallFailed
			},
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "post-install failed")
	})

	t.Run("returns error when expected artifacts missing", func(t *testing.T) {
		t.Parallel()

		// Prepare valid archive
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		CreateTestTarGz(t, tarPath, map[string]string{"repo-1.0.0/f": "c"})

		content, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(content)
			if err != nil {
				panic(err)
			}
		}))
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:          "repo",
			SourceURLTemplate: server.URL,
			BuildVersion:      func(_ context.Context, _, _, _ string) error { return nil },
			ExpectedArtifacts: []string{"bin/missing"},
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "install artifact missing")
	})
}

func TestSourceBuildPluginLatestStable(t *testing.T) {
	t.Parallel()

	t.Run("returns latest stable version", func(t *testing.T) {
		t.Parallel()

		srv := githubmock.NewServer()
		defer srv.Close()

		srv.AddReleases("o", "r", []string{"v1.0.0", "v1.1.0", "v2.0.0-rc1"})

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoOwner: "o",
			RepoName:  "r",
		})
		plugin.WithGithubClient(github.NewClientWithHTTP(srv.HTTPServer.Client(), srv.URL()))

		version, err := plugin.LatestStable(t.Context(), "")
		require.NoError(t, err)
		require.Equal(t, "1.1.0", version)
	})

	t.Run("returns error when no versions found", func(t *testing.T) {
		t.Parallel()

		srv := githubmock.NewServer()
		defer srv.Close()

		srv.AddReleases("o", "r", nil)

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoOwner: "o",
			RepoName:  "r",
		})
		plugin.WithGithubClient(github.NewClientWithHTTP(srv.HTTPServer.Client(), srv.URL()))

		_, err := plugin.LatestStable(t.Context(), "")
		require.ErrorIs(t, err, asdf.ErrSourceBuildNoVersionsFoundForTests())
	})

	t.Run("returns error when no matching versions", func(t *testing.T) {
		t.Parallel()

		srv := githubmock.NewServer()
		defer srv.Close()

		srv.AddReleases("o", "r", []string{"v1.0.0"})

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoOwner: "o",
			RepoName:  "r",
		})
		plugin.WithGithubClient(github.NewClientWithHTTP(srv.HTTPServer.Client(), srv.URL()))

		_, err := plugin.LatestStable(t.Context(), "2.")
		require.ErrorIs(t, err, asdf.ErrSourceBuildNoVersionsMatchingForTests())
	})
}

func TestSourceBuildPluginExtractSource(t *testing.T) {
	t.Parallel()

	t.Run("returns error for unsupported archive type", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			ArchiveType: "rar", // Unsupported
		})

		// Create dummy archive
		tempDir := t.TempDir()
		archivePath := filepath.Join(tempDir, "archive.rar")
		require.NoError(t, os.WriteFile(archivePath, []byte("dummy"), 0o600))

		// Access private method via Install (indirectly) or use reflection?
		// Since we can't access private method, we test via Install failing at extract
		// But wait, Install calls extractSource.
		// We can test this by setting ArchiveType in config and running Install with a dummy download.

		plugin.Config.BuildVersion = func(_ context.Context, _, _, _ string) error { return nil }
		plugin.Config.DownloadFile = func(_ context.Context, _, dest string) error {
			return os.WriteFile(dest, []byte("dummy"), 0o600)
		}
		plugin.Config.SourceURLTemplate = "http://dummy"

		err := plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported archive type")
	})

	t.Run("returns error when archive missing", func(t *testing.T) {
		t.Parallel()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{})

		// We can't easily call private extractSource.
		// However, Install guarantees download happens or exists.
		// If SkipDownload is true, it expects archive to be there.

		plugin.Config.SkipDownload = true
		plugin.Config.BuildVersion = func(_ context.Context, _, _, _ string) error { return nil }

		err := plugin.Install(t.Context(), "1.0.0", t.TempDir(), t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "source archive missing")
	})

	t.Run("returns error when extracted dir missing", func(t *testing.T) {
		t.Parallel()

		// Case where extraction succeeds but the expected directory is not found
		tempDir := t.TempDir()
		tarPath := filepath.Join(tempDir, "archive.tar.gz")
		// Archive extracts to "wrong-dir"
		CreateTestTarGz(t, tarPath, map[string]string{"wrong-dir/file": "content"})

		content, err := os.ReadFile(tarPath)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(content)
			if err != nil {
				t.Errorf("writing archive response: %v", err)
			}
		}))
		defer server.Close()

		plugin := asdf.NewSourceBuildPlugin(&asdf.SourceBuildPluginConfig{
			RepoName:                 "repo",
			ExtractedDirNameTemplate: "repo-{{.Version}}", // Expects repo-1.0.0
			SourceURLTemplate:        server.URL,
			BuildVersion:             func(_ context.Context, _, _, _ string) error { return nil },
		})

		err = plugin.Install(t.Context(), "1.0.0", "", t.TempDir())
		require.Error(t, err)
		require.Contains(t, err.Error(), "extracted source directory missing")
	})
}
