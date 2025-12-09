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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// gcloudTestFixture provides plugin instances for testing.
type gcloudTestFixture struct {
	server       *httptest.Server
	plugin       *Plugin
	testdataPath string
	versions     []string
}

// newGcloudTestFixture creates a mock test fixture for the plugin.
func newGcloudTestFixture() *gcloudTestFixture {
	fixture := &gcloudTestFixture{
		versions: nil,
	}

	_, file, _, _ := runtime.Caller(0)

	fixture.testdataPath = filepath.Join(filepath.Dir(file), "testdata")

	fixture.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		items := make([]gcsObject, 0, len(fixture.versions))
		for _, version := range fixture.versions {
			platform := "linux"
			if runtime.GOOS == "darwin" {
				platform = "darwin"
			}

			arch := "x86_64"
			if runtime.GOARCH == "arm64" {
				arch = "arm"
			}

			items = append(items, gcsObject{
				Name: fmt.Sprintf("google-cloud-sdk-%s-%s-%s.tar.gz", version, platform, arch),
			})
		}

		resp := gcsResponse{Items: items}

		writer.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(writer).Encode(resp) //nolint:errcheck // test mock
	}))

	fixture.plugin = NewWithURL(fixture.server.URL)

	return fixture
}

// Close shuts down the mock server.
func (fixture *gcloudTestFixture) Close() {
	if fixture.server != nil {
		fixture.server.Close()
	}
}

// SetupVersions registers the given versions with the mock server.
func (fixture *gcloudTestFixture) SetupVersions(versions []string) {
	fixture.versions = versions
}

// SetupTagsFromGoldie reads versions from goldie test data and registers them.
// This ensures tests use the same versions as the goldie snapshots.
func (fixture *gcloudTestFixture) SetupTagsFromGoldie() error {
	versions, err := testutil.ReadGoldieVersions(fixture.testdataPath, "gcloud_list_all.golden")
	if err != nil {
		return err
	}

	fixture.SetupVersions(versions)

	return nil
}

// GoldieVersions returns the versions from the goldie test data.
func (fixture *gcloudTestFixture) GoldieVersions() ([]string, error) {
	return testutil.ReadGoldieVersions(fixture.testdataPath, "gcloud_list_all.golden")
}

// GoldieLatest returns the latest version from the goldie test data.
func (fixture *gcloudTestFixture) GoldieLatest() (string, error) {
	return testutil.ReadGoldieLatest(fixture.testdataPath, "gcloud_latest_stable.golden")
}

// GoldieFilterPattern returns a filter pattern based on goldie versions.
func (fixture *gcloudTestFixture) GoldieFilterPattern() (string, error) {
	versions, err := fixture.GoldieVersions()
	if err != nil {
		return "", err
	}

	return testutil.GenerateFilterPattern(versions), nil
}

// GoldieFilesExist returns true if the required goldie files exist.
func (fixture *gcloudTestFixture) GoldieFilesExist() bool {
	return testutil.GoldieFileExists(fixture.testdataPath, "gcloud_list_all.golden") &&
		testutil.GoldieFileExists(fixture.testdataPath, "gcloud_latest_stable.golden")
}
