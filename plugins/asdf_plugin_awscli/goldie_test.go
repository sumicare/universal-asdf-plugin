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
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// TestHelpGoldie verifies that the plugin help output matches the golden snapshot.
func TestHelpGoldie(t *testing.T) {
	p := New()
	h := p.Help()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "awscli_help", []byte(strings.Join([]string{h.Overview, h.Deps, h.Config, h.Links}, "\n\n")))
}

// TestListBinPathsGoldie verifies that ListBinPaths output matches the golden snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "awscli_bin_paths", []byte(p.ListBinPaths()))
}

// TestListLegacyFilenamesGoldie verifies that ListLegacyFilenames output matches the golden snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	p := New()
	files := p.ListLegacyFilenames()

	var output string
	if files != nil {
		output = strings.Join(files, "\n")
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "awscli_legacy_filenames", []byte(output))
}

// TestExecEnvGoldie verifies that ExecEnv output matches the golden snapshot.
func TestExecEnvGoldie(t *testing.T) {
	p := New()
	env := p.ExecEnv("/tmp/install")

	output := make([]string, 0, len(env))
	for k, v := range env {
		output = append(output, k+"="+v)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "awscli_exec_env", []byte(strings.Join(output, "\n")))
}

// TestListAllGoldie verifies that ListAll output matches the golden snapshot.
func TestListAllGoldie(t *testing.T) {
	if asdf.IsOnline() {
		p := New()

		versions, err := p.ListAll(t.Context())
		if err != nil {
			t.Fatalf("ListAll failed: %v", err)
		}

		goldieRecorder := goldie.New(t)
		goldieRecorder.Assert(t, "awscli_list_all", []byte(strings.Join(versions, "\n")))

		return
	}

	testdataPath := testutil.GoldieTestDataPath(t)

	if !testutil.GoldieFileExists(testdataPath, "awscli_list_all.golden") {
		t.Skip("awscli_list_all.golden not found - run with ONLINE=1 to create")
	}

	fixture := newAwscliTestFixture()
	defer fixture.Close()

	err := fixture.SetupTagsFromGoldie()
	if err != nil {
		t.Fatalf("failed to setup tags from goldie: %v", err)
	}

	versions, err := fixture.plugin.ListAll(t.Context())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "awscli_list_all", []byte(strings.Join(versions, "\n")))
}

// TestLatestStableGoldie verifies that LatestStable behavior matches the golden snapshots for different queries.
func TestLatestStableGoldie(t *testing.T) {
	if asdf.IsOnline() {
		p := New()

		version, err := p.LatestStable(t.Context(), "")
		if err != nil {
			t.Fatalf("LatestStable failed: %v", err)
		}

		goldieRecorder := goldie.New(t)
		goldieRecorder.Assert(t, "awscli_latest_stable", []byte(version))

		return
	}

	testdataPath := testutil.GoldieTestDataPath(t)

	if !testutil.GoldieFileExists(testdataPath, "awscli_list_all.golden") {
		t.Skip("awscli_list_all.golden not found - run with ONLINE=1 to create")
	}

	if !testutil.GoldieFileExists(testdataPath, "awscli_latest_stable.golden") {
		t.Skip("awscli_latest_stable.golden not found - run with ONLINE=1 to create")
	}

	fixture := newAwscliTestFixture()
	defer fixture.Close()

	err := fixture.SetupTagsFromGoldie()
	if err != nil {
		t.Fatalf("failed to setup tags from goldie: %v", err)
	}

	version, err := fixture.plugin.LatestStable(t.Context(), "")
	if err != nil {
		t.Fatalf("LatestStable failed: %v", err)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "awscli_latest_stable", []byte(version))
}
