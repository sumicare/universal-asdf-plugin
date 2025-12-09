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

package asdf_plugin_python

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"

	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// TestHelpGoldie verifies that the plugin help output matches the golden snapshot.
func TestHelpGoldie(t *testing.T) {
	RegisterTestingT(t)

	p := New()
	h := p.Help()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "python_help", []byte(strings.Join([]string{h.Overview, h.Deps, h.Config, h.Links}, "\n\n")))
}

// TestListBinPathsGoldie verifies that ListBinPaths output matches the golden snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	RegisterTestingT(t)

	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "python_bin_paths", []byte(p.ListBinPaths()))
}

// TestListLegacyFilenamesGoldie verifies that ListLegacyFilenames output matches the golden snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	RegisterTestingT(t)

	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "python_legacy_filenames", []byte(strings.Join(p.ListLegacyFilenames(), "\n")))
}

// TestListAllFromPythonBuildGoldie verifies that ListAllFromPythonBuild output matches the goldie snapshot.
func TestListAllFromPythonBuildGoldie(t *testing.T) {
	RegisterTestingT(t)

	if _, err := exec.LookPath("python-build"); err != nil {
		t.Skip("python-build not available")
	}

	p := New()

	versions, err := p.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	goldie.New(t).Assert(t, "python_list_all", []byte(strings.Join(versions[:min(20, len(versions))], "\n")))
}

// TestListAllFromFTPGoldie verifies that ListAllFromFTP output matches the goldie snapshot.
func TestListAllFromFTPGoldie(t *testing.T) {
	RegisterTestingT(t)

	testdataPath := testutil.GoldieTestDataPath(t)
	if !testutil.GoldieFileExists(testdataPath, "python_list_all_ftp.golden") {
		t.Skip("python_list_all_ftp.golden not found - run with ONLINE=1 to create")
	}

	fixture := newPythonTestFixture()
	defer fixture.Close()

	fixture.SetupVersion("3.9.0")
	fixture.SetupVersion("3.10.0")
	fixture.SetupVersion("3.11.0")
	fixture.SetupVersion("3.12.0")

	versions, err := fixture.plugin.ListAllFromFTP(t.Context())
	if err != nil {
		t.Fatalf("ListAllFromFTP failed: %v", err)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "python_list_all_ftp", []byte(strings.Join(versions, "\n")))
}

// TestListAllFromGitHubGoldie verifies that ListAllFromGitHub output matches the goldie snapshot.
func TestListAllFromGitHubGoldie(t *testing.T) {
	RegisterTestingT(t)

	fixture := newPythonTestFixture()
	defer fixture.Close()

	ghServer := mock.NewServer("python", "cpython")
	defer ghServer.Close()

	githubClient := github.NewClientWithHTTP(ghServer.Client(), ghServer.URL())

	plugin := NewWithURLs("http://ftp-mock", githubClient)

	ghServer.RegisterTag("v3.9.0")
	ghServer.RegisterTag("v3.9.1")
	ghServer.RegisterTag("v3.10.0")
	ghServer.RegisterTag("v3.10.1")
	ghServer.RegisterTag("v3.11.0")
	ghServer.RegisterTag("v3.11.1")
	ghServer.RegisterTag("v3.12.0")

	versions, err := plugin.ListAllFromGitHub(t.Context())
	if err != nil {
		t.Fatalf("ListAllFromGitHub failed: %v", err)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "python_list_all_github", []byte(strings.Join(versions, "\n")))
}

// TestLatestStableGoldie verifies that LatestStable behavior matches the golden snapshots for different queries.
func TestLatestStableGoldie(t *testing.T) {
	RegisterTestingT(t)

	testdataPath := testutil.GoldieTestDataPath(t)

	required := []string{
		"python_latest_stable.golden",
		"python_latest_stable_3_11.golden",
		"python_latest_stable_3_10.golden",
	}
	for _, file := range required {
		if !testutil.GoldieFileExists(testdataPath, file) {
			t.Skip(file + " not found - run with ONLINE=1 to create")
		}
	}

	fixture := newPythonTestFixture()
	defer fixture.Close()

	fixture.SetupVersion("3.10.0")
	fixture.SetupVersion("3.10.5")
	fixture.SetupVersion("3.11.0")
	fixture.SetupVersion("3.11.5")
	fixture.SetupVersion("3.12.0")

	tests := []struct {
		query string
		name  string
	}{
		{"", "python_latest_stable"},
		{"3.11", "python_latest_stable_3_11"},
		{"3.10", "python_latest_stable_3_10"},
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
