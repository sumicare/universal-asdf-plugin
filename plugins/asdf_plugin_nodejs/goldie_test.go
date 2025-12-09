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

package asdf_plugin_nodejs

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"

	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
	githubmock "github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

// TestHelpGoldie verifies that Help output matches the goldie snapshot.
func TestHelpGoldie(t *testing.T) {
	RegisterTestingT(t)

	p := New()
	h := p.Help()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "nodejs_help", []byte(strings.Join([]string{h.Overview, h.Deps, h.Config, h.Links}, "\n\n")))
}

// TestListBinPathsGoldie verifies that ListBinPaths output matches the goldie snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	RegisterTestingT(t)

	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "nodejs_bin_paths", []byte(p.ListBinPaths()))
}

// TestListLegacyFilenamesGoldie verifies that ListLegacyFilenames output matches the goldie snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	RegisterTestingT(t)

	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "nodejs_legacy_filenames", []byte(strings.Join(p.ListLegacyFilenames(), "\n")))
}

// TestLTSCodenamesGoldie verifies that the LTS codenames JSON matches the goldie snapshot.
func TestLTSCodenamesGoldie(t *testing.T) {
	RegisterTestingT(t)

	codenames := map[string]string{"Hydrogen": "18.12.0", "Iron": "20.9.0"}

	data, _ := json.MarshalIndent(codenames, "", "  ") //nolint:errcheck // this is a safe value
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "lts_codenames", data)
}

// TestGetLTSCodenames verifies that GetLTSCodenames returns the expected LTS
// codename map when backed by the mocked Node.js index.
func TestGetLTSCodenames(t *testing.T) {
	RegisterTestingT(t)

	fixture := newNodeMockFixture()
	defer fixture.Close()

	fixture.SetupVersion("18.12.0", "linux", "x64", "Hydrogen")
	fixture.SetupVersion("20.9.0", "linux", "x64", "Iron")
	fixture.SetupVersion("21.0.0", "linux", "x64", false)

	codenames, err := fixture.plugin.GetLTSCodenames(t.Context())
	if err != nil {
		t.Fatalf("GetLTSCodenames failed: %v", err)
	}

	Expect(codenames).To(HaveKeyWithValue("Hydrogen", "18.12.0"))
	Expect(codenames).To(HaveKeyWithValue("Iron", "20.9.0"))
	Expect(codenames).NotTo(HaveKey("NonExistent"))
}

// TestListAllFromGitHubMock verifies that ListAllFromGitHub parses versions
// from a mocked GitHub releases feed without requiring real network access.
func TestListAllFromGitHubMock(t *testing.T) {
	RegisterTestingT(t)

	server := githubmock.NewServer()
	defer server.Close()

	server.AddReleases("nodejs", "node", []string{"v18.12.0", "v20.9.0"})

	client := github.NewClientWithHTTP(server.HTTPServer.Client(), server.URL())
	plugin := newWithGitHubClient(client)

	versions, err := plugin.ListAllFromGitHub(t.Context())
	if err != nil {
		t.Fatalf("ListAllFromGitHub failed: %v", err)
	}

	Expect(versions).To(ContainElements("18.12.0", "20.9.0"))
}

// TestListAllFromNodeBuildGoldie verifies that versions from node-build match the goldie snapshot.
func TestListAllFromNodeBuildGoldie(t *testing.T) {
	RegisterTestingT(t)

	if _, err := exec.LookPath("node-build"); err != nil {
		t.Skip("node-build not available")
	}

	p := New()

	versions, err := p.listAllFromNodeBuild(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	goldie.New(t).Assert(t, "nodejs_list_all", []byte(strings.Join(versions[:min(20, len(versions))], "\n")))
}

// TestListAllFromAPIGoldie verifies that versions from the mocked Node.js API match the goldie snapshot.
func TestListAllFromAPIGoldie(t *testing.T) {
	RegisterTestingT(t)

	testdataPath := testutil.GoldieTestDataPath(t)
	if !testutil.GoldieFileExists(testdataPath, "nodejs_list_all_api.golden") {
		t.Skip("nodejs_list_all_api.golden not found - run with ONLINE=1 to create")
	}

	fixture := newNodeMockFixture()
	defer fixture.Close()

	fixture.SetupVersion("18.12.0", "linux", "x64", "Hydrogen")
	fixture.SetupVersion("18.12.1", "linux", "x64", "Hydrogen")
	fixture.SetupVersion("18.13.0", "linux", "x64", "Hydrogen")
	fixture.SetupVersion("20.9.0", "linux", "x64", "Iron")
	fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")
	fixture.SetupVersion("20.11.0", "linux", "x64", "Iron")
	fixture.SetupVersion("21.0.0", "linux", "x64", false)
	fixture.SetupVersion("21.1.0", "linux", "x64", false)

	versions, err := fixture.plugin.listAllFromAPI(t.Context())
	if err != nil {
		t.Fatalf("listAllFromAPI failed: %v", err)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "nodejs_list_all_api", []byte(strings.Join(versions, "\n")))
}

// TestLatestStableGoldie verifies that LatestStable behavior matches the goldie snapshots for different queries.
func TestLatestStableGoldie(t *testing.T) {
	RegisterTestingT(t)

	testdataPath := testutil.GoldieTestDataPath(t)

	required := []string{
		"nodejs_latest_stable.golden",
		"nodejs_latest_stable_20.golden",
		"nodejs_latest_stable_18.golden",
		"nodejs_latest_lts.golden",
		"nodejs_latest_lts_iron.golden",
		"nodejs_latest_lts_hydrogen.golden",
	}
	for _, file := range required {
		if !testutil.GoldieFileExists(testdataPath, file) {
			t.Skip(file + " not found - run with ONLINE=1 to create")
		}
	}

	fixture := newNodeMockFixture()
	defer fixture.Close()

	fixture.SetupVersion("18.12.0", "linux", "x64", "Hydrogen")
	fixture.SetupVersion("18.12.1", "linux", "x64", "Hydrogen")
	fixture.SetupVersion("20.9.0", "linux", "x64", "Iron")
	fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")
	fixture.SetupVersion("21.0.0", "linux", "x64", false)
	fixture.SetupVersion("21.1.0", "linux", "x64", false)

	tests := []struct {
		query string
		name  string
	}{
		{"", "nodejs_latest_stable"},
		{"20", "nodejs_latest_stable_20"},
		{"18", "nodejs_latest_stable_18"},
		{"lts", "nodejs_latest_lts"},
		{"lts/iron", "nodejs_latest_lts_iron"},
		{"lts/hydrogen", "nodejs_latest_lts_hydrogen"},
	}

	goldieRecorder := goldie.New(t)
	for i := range tests {
		version, err := fixture.plugin.LatestStable(t.Context(), tests[i].query)
		if err != nil {
			t.Fatalf("LatestStable(%q) failed: %v", tests[i].query, err)
		}

		goldieRecorder.Assert(t, tests[i].name, []byte(version))
	}
}
