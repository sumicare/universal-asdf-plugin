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
	"fmt"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

type (
	// PluginTestConfig configures plugin test fixtures for both Ginkgo and Goldie tests.
	// It references the plugin's BinaryPluginConfig and provides factory functions.
	PluginTestConfig struct {
		Config              *asdf.BinaryPluginConfig
		NewPlugin           func() asdf.Plugin
		NewPluginWithClient func(client *github.Client) asdf.Plugin
		TestdataPath        string
		ForceMock           bool
	}

	// BinaryPluginTestFixture provides a universal test fixture for binary plugins.
	// It handles mock server setup, goldie file operations, and version management.
	BinaryPluginTestFixture struct {
		Plugin       asdf.Plugin
		Server       *mock.Server
		Config       *PluginTestConfig
		TestdataPath string
	}
)

// NewBinaryPluginTestFixture creates a new test fixture for a binary plugin.
// The callerSkip parameter is deprecated and ignored; use cfg.TestdataPath instead.
func NewBinaryPluginTestFixture(cfg *PluginTestConfig, _ int) *BinaryPluginTestFixture {
	return NewBinaryPluginTestFixtureWithMode(cfg, false)
}

// NewBinaryPluginTestFixtureWithMode creates a test fixture with explicit mode control.
// When forceMock is true, creates a mock fixture even in ONLINE mode (for mock-specific tests).
func NewBinaryPluginTestFixtureWithMode(cfg *PluginTestConfig, forceMock bool) *BinaryPluginTestFixture {
	fixture := &BinaryPluginTestFixture{
		Config:       cfg,
		TestdataPath: cfg.TestdataPath,
	}

	useMock := forceMock || !asdf.IsOnline()

	if useMock {
		fixture.Server = mock.NewServer(cfg.Config.RepoOwner, cfg.Config.RepoName)

		githubClient := github.NewClientWithHTTP(fixture.Server.Client(), fixture.Server.URL())

		fixture.Plugin = cfg.NewPluginWithClient(githubClient)

		if bp, ok := fixture.Plugin.(*asdf.BinaryPlugin); ok {
			versionPrefix := cfg.Config.VersionPrefix
			if versionPrefix == "" {
				versionPrefix = "v"
			}

			bp.Config.DownloadURLTemplate = fixture.Server.URL() + "/{{.RepoOwner}}/{{.RepoName}}/releases/download/" + versionPrefix + "{{.Version}}/{{.FileName}}"
		}
	} else {
		fixture.Plugin = cfg.NewPlugin()
	}

	return fixture
}

// Close shuts down the mock server if one was created.
func (fixture *BinaryPluginTestFixture) Close() {
	if fixture.Server != nil {
		fixture.Server.Close()
	}
}

// SetupVersion registers a mock download and tag for the given version/platform/arch.
func (fixture *BinaryPluginTestFixture) SetupVersion(version, platform, arch string) {
	cfg := fixture.Config.Config

	mappedPlatform := platform
	if cfg.OsMap != nil {
		if mapped, ok := cfg.OsMap[platform]; ok {
			mappedPlatform = mapped
		}
	}

	mappedArch := arch
	if cfg.ArchMap != nil {
		if mapped, ok := cfg.ArchMap[arch]; ok {
			mappedArch = mapped
		}
	}

	filename := cfg.FileNameTemplate

	filename = strings.ReplaceAll(filename, "{{.Version}}", version)
	filename = strings.ReplaceAll(filename, "{{.Platform}}", mappedPlatform)
	filename = strings.ReplaceAll(filename, "{{.Arch}}", mappedArch)
	filename = strings.ReplaceAll(filename, "{{.BinaryName}}", cfg.BinaryName)

	tagPrefix := cfg.VersionPrefix
	if tagPrefix == "" {
		tagPrefix = "v"
	}

	path := fmt.Sprintf("/%s/%s/releases/download/%s%s/%s",
		cfg.RepoOwner, cfg.RepoName, tagPrefix, version, filename)

	mockContent := "#!/bin/sh\necho 'mock binary'\n"

	binaryPath := cfg.BinaryName

	switch cfg.ArchiveType {
	case "gz":
		fixture.Server.RegisterGzDownload(path, mockContent)
	case "tar.gz":
		fixture.Server.RegisterTarGzDownload(path, map[string]string{binaryPath: mockContent})
	case "tar.xz":
		fixture.Server.RegisterTarXzDownload(path, map[string]string{binaryPath: mockContent})
	case "zip":
		fixture.Server.RegisterZipDownload(path, map[string]string{binaryPath: mockContent})
	default:
		fixture.Server.RegisterDownload(path)
	}

	fixture.Server.RegisterTag(tagPrefix + version)
}

// SetupTags registers the given tags with the mock server.
func (fixture *BinaryPluginTestFixture) SetupTags(tags []string) {
	for _, tag := range tags {
		fixture.Server.RegisterTag(tag)
	}
}

// SetupTagsFromGoldie reads versions from goldie test data and registers them as tags.
func (fixture *BinaryPluginTestFixture) SetupTagsFromGoldie() error {
	versions, err := ReadGoldieVersions(fixture.TestdataPath, fixture.ListAllGoldenFile())
	if err != nil {
		return err
	}

	tagPrefix := fixture.Config.Config.VersionPrefix
	if tagPrefix == "" {
		tagPrefix = "v"
	}

	tags := make([]string, 0, len(versions))
	for i := range versions {
		tags = append(tags, tagPrefix+versions[i])
	}

	fixture.SetupTags(tags)

	return nil
}

// GoldieVersions returns the versions from the goldie test data.
func (fixture *BinaryPluginTestFixture) GoldieVersions() ([]string, error) {
	return ReadGoldieVersions(fixture.TestdataPath, fixture.ListAllGoldenFile())
}

// GoldieLatest returns the latest version from the goldie test data.
func (fixture *BinaryPluginTestFixture) GoldieLatest() (string, error) {
	return ReadGoldieLatest(fixture.TestdataPath, fixture.LatestStableGoldenFile())
}

// GoldieFilterPattern returns a filter pattern based on goldie versions.
func (fixture *BinaryPluginTestFixture) GoldieFilterPattern() (string, error) {
	versions, err := fixture.GoldieVersions()
	if err != nil {
		return "", err
	}

	return GenerateFilterPattern(versions), nil
}

// GoldieFilesExist returns true if the required goldie files exist.
func (fixture *BinaryPluginTestFixture) GoldieFilesExist() bool {
	return GoldieFileExists(fixture.TestdataPath, fixture.ListAllGoldenFile()) &&
		GoldieFileExists(fixture.TestdataPath, fixture.LatestStableGoldenFile())
}

// ListAllGoldenFile returns the golden file name for list_all (public accessor).
func (fixture *BinaryPluginTestFixture) ListAllGoldenFile() string {
	return fixture.GoldenPrefix() + "_list_all.golden"
}

// LatestStableGoldenFile returns the golden file name for latest_stable (public accessor).
func (fixture *BinaryPluginTestFixture) LatestStableGoldenFile() string {
	return fixture.GoldenPrefix() + "_latest_stable.golden"
}

// GoldenPrefix returns the prefix for golden files (public accessor).
func (fixture *BinaryPluginTestFixture) GoldenPrefix() string {
	return strings.ReplaceAll(fixture.Config.Config.Name, "-", "_")
}
