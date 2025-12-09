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

package asdf_plugin_go

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"

	"github.com/sumicare/universal-asdf-plugin/plugins/github"
	githubmock "github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

// TestHelpGoldie verifies that the Go plugin help output matches the goldie snapshot.
func TestHelpGoldie(t *testing.T) {
	p := New()
	h := p.Help()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "go_help", []byte(strings.Join([]string{h.Overview, h.Deps, h.Config, h.Links}, "\n\n")))
}

// TestListBinPathsGoldie verifies that ListBinPaths output matches the golden snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "go_bin_paths", []byte(p.ListBinPaths()))
}

// TestListLegacyFilenamesGoldie verifies that ListLegacyFilenames output matches the golden snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "go_legacy_filenames", []byte(strings.Join(p.ListLegacyFilenames(), "\n")))
}

// TestVersionParsingGoldie verifies that Go version tags are parsed and sorted
// as expected and match the golden snapshot.
func TestVersionParsingGoldie(t *testing.T) {
	tags := []string{"go1.2.2", "go1.3", "go1.3.1", "go1.20.0", "go1.21.0", "go1.21rc1"}
	versions := filterOldVersions(parseGoTags(tags))
	sortGoVersions(versions)

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "go_versions", []byte(strings.Join(versions, "\n")))
}

// TestListAllGoldie verifies that ListAll output matches the golden snapshot.
func TestListAllGoldie(t *testing.T) {
	server := githubmock.NewServer()
	defer server.Close()

	githubClient := github.NewClientWithHTTP(&http.Client{}, server.URL())
	mockPlugin := NewWithURLs(goDownloadURL, githubClient)

	server.AddTags("golang", "go", []string{
		"go1.19.0", "go1.19.1", "go1.19.13",
		"go1.20.0", "go1.20.1", "go1.20.12",
		"go1.21.0", "go1.21.1", "go1.21.5",
		"go1.22.0", "go1.22.1",
	})

	versions, err := mockPlugin.ListAll(t.Context())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "go_list_all", []byte(strings.Join(versions, "\n")))
}

// TestExecEnvGoldie verifies that ExecEnv output matches the golden snapshot.
func TestExecEnvGoldie(t *testing.T) {
	t.Setenv("GOROOT", "")
	t.Setenv("GOPATH", "")
	t.Setenv("GOBIN", "")

	p := New()
	env := p.ExecEnv("/tmp/install")

	output := make([]string, 0, len(env))
	for k, v := range env {
		output = append(output, k+"="+v)
	}

	sort.Strings(output)

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "go_exec_env", []byte(strings.Join(output, "\n")))
}

// TestLatestStableGoldie verifies that LatestStable behavior matches the golden snapshots for different queries.
func TestLatestStableGoldie(t *testing.T) {
	server := githubmock.NewServer()
	defer server.Close()

	githubClient := github.NewClientWithHTTP(&http.Client{}, server.URL())
	mockPlugin := NewWithURLs(goDownloadURL, githubClient)

	server.AddTags("golang", "go", []string{
		"go1.20.0", "go1.20.12",
		"go1.21.0", "go1.21.5", "go1.21rc1",
		"go1.22.0", "go1.22.1",
	})

	tests := []struct {
		query string
		name  string
	}{
		{"", "go_latest_stable"},
		{"1.21", "go_latest_stable_1_21"},
		{"1.20", "go_latest_stable_1_20"},
	}

	goldieRecorder := goldie.New(t)
	for i := range tests {
		version, err := mockPlugin.LatestStable(t.Context(), tests[i].query)
		if err != nil {
			t.Fatalf("LatestStable(%q) failed: %v", tests[i].query, err)
		}

		goldieRecorder.Assert(t, tests[i].name, []byte(version))
	}
}
