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

package asdf_plugin_awscli

import (
	"path/filepath"
	"runtime"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// awscliTestFixture provides plugin instances for testing.
type awscliTestFixture struct {
	server       *mock.Server
	plugin       *Plugin
	testdataPath string
}

// newAwscliTestFixture creates a test fixture for the plugin using the mock server.
// It delegates to newAwscliMockFixture so behavior is consistent in both offline and ONLINE modes.
func newAwscliTestFixture() *awscliTestFixture {
	return newAwscliMockFixture()
}

// newAwscliMockFixture creates a mock test fixture for the plugin.
func newAwscliMockFixture() *awscliTestFixture {
	fixture := &awscliTestFixture{}

	_, file, _, _ := runtime.Caller(0)

	fixture.testdataPath = filepath.Join(filepath.Dir(file), "testdata")

	fixture.server = mock.NewServer("aws", "aws-cli")

	githubClient := github.NewClientWithHTTP(fixture.server.Client(), fixture.server.URL())

	fixture.plugin = NewWithClient(githubClient)

	return fixture
}

// Close shuts down the mock server if one was created.
func (fixture *awscliTestFixture) Close() {
	if fixture.server != nil {
		fixture.server.Close()
	}
}

// SetupTags registers the given tags with the mock server for mock testing.
func (fixture *awscliTestFixture) SetupTags(tags []string) {
	if fixture.server == nil {
		return
	}

	if tags == nil {
		fixture.server.ClearTags()
		return
	}

	fixture.server.ClearTags()

	for _, tag := range tags {
		fixture.server.RegisterTag(tag)
	}
}

// SetupTagsFromGoldie reads versions from goldie test data and registers them as tags.
// This ensures tests use the same versions as the goldie snapshots.
func (fixture *awscliTestFixture) SetupTagsFromGoldie() error {
	versions, err := testutil.ReadGoldieVersions(fixture.testdataPath, "awscli_list_all.golden")
	if err != nil {
		return err
	}

	tags := testutil.VersionsToTags(versions, false)
	fixture.SetupTags(tags)

	return nil
}

// GoldieVersions returns the versions from the goldie test data.
func (fixture *awscliTestFixture) GoldieVersions() ([]string, error) {
	return testutil.ReadGoldieVersions(fixture.testdataPath, "awscli_list_all.golden")
}

// GoldieLatest returns the latest version from the goldie test data.
func (fixture *awscliTestFixture) GoldieLatest() (string, error) {
	return testutil.ReadGoldieLatest(fixture.testdataPath, "awscli_latest_stable.golden")
}

// GoldieFilterPattern returns a filter pattern based on goldie versions.
func (fixture *awscliTestFixture) GoldieFilterPattern() (string, error) {
	versions, err := fixture.GoldieVersions()
	if err != nil {
		return "", err
	}

	return testutil.GenerateFilterPattern(versions), nil
}

// GoldieFilesExist returns true if the required goldie files exist.
func (fixture *awscliTestFixture) GoldieFilesExist() bool {
	return testutil.GoldieFileExists(fixture.testdataPath, "awscli_list_all.golden") &&
		testutil.GoldieFileExists(fixture.testdataPath, "awscli_latest_stable.golden")
}
