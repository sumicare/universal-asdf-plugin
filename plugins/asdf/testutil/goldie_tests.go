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
	"flag"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// goldenPrefix returns the normalized plugin name for golden file naming.
// It replaces hyphens with underscores to match file naming conventions.
func goldenPrefix(cfg *PluginTestConfig) string {
	return strings.ReplaceAll(cfg.Config.Name, "-", "_")
}

// isUpdateMode reports whether tests are running with the -update flag enabled.
func isUpdateMode() bool {
	f := flag.Lookup("update")
	if f == nil {
		return false
	}

	return f.Value.String() == "true"
}

// RunHelpGoldie runs the Help goldie test.
func RunHelpGoldie(t *testing.T, cfg *PluginTestConfig) {
	t.Helper()

	p := cfg.NewPlugin()
	h := p.Help()
	g := goldie.New(t)
	g.Assert(t, goldenPrefix(cfg)+"_help", []byte(strings.Join([]string{h.Overview, h.Deps, h.Config, h.Links}, "\n\n")))
}

// RunListBinPathsGoldie runs the ListBinPaths goldie test.
func RunListBinPathsGoldie(t *testing.T, cfg *PluginTestConfig) {
	t.Helper()

	p := cfg.NewPlugin()
	g := goldie.New(t)
	g.Assert(t, goldenPrefix(cfg)+"_bin_paths", []byte(p.ListBinPaths()))
}

// RunListLegacyFilenamesGoldie runs the ListLegacyFilenames goldie test.
func RunListLegacyFilenamesGoldie(t *testing.T, cfg *PluginTestConfig) {
	t.Helper()

	p := cfg.NewPlugin()
	g := goldie.New(t)
	g.Assert(t, goldenPrefix(cfg)+"_legacy_filenames", []byte(strings.Join(p.ListLegacyFilenames(), "\n")))
}

// RunExecEnvGoldie runs the ExecEnv goldie test.
func RunExecEnvGoldie(t *testing.T, cfg *PluginTestConfig) {
	t.Helper()

	p := cfg.NewPlugin()
	env := p.ExecEnv("/tmp/install")

	output := make([]string, 0, len(env))
	for k, v := range env {
		output = append(output, k+"="+v)
	}

	g := goldie.New(t)
	g.Assert(t, goldenPrefix(cfg)+"_exec_env", []byte(strings.Join(output, "\n")))
}

// RunListAllGoldie runs the ListAll goldie test.
func RunListAllGoldie(t *testing.T, cfg *PluginTestConfig) {
	t.Helper()

	testdataPath := cfg.TestdataPath
	prefix := goldenPrefix(cfg)
	listAllFile := prefix + "_list_all.golden"

	if isUpdateMode() {
		p := cfg.NewPlugin()

		versions, err := p.ListAll(t.Context())
		if err != nil {
			t.Fatalf("ListAll failed: %v", err)
		}

		g := goldie.New(t)
		g.Assert(t, prefix+"_list_all", []byte(strings.Join(versions, "\n")))

		return
	}

	server := mock.NewServer(cfg.Config.RepoOwner, cfg.Config.RepoName)
	defer server.Close()

	githubClient := github.NewClientWithHTTP(server.Client(), server.URL())
	mockPlugin := cfg.NewPluginWithClient(githubClient)

	if !GoldieFileExists(testdataPath, listAllFile) {
		t.Skip(listAllFile + " not found - run with ONLINE=1 to create")
	}

	versions, err := ReadGoldieVersions(testdataPath, listAllFile)
	if err != nil {
		t.Fatalf("failed to read goldie versions: %v", err)
	}

	tagPrefix := cfg.Config.VersionPrefix
	if tagPrefix == "" {
		tagPrefix = "v"
	}

	for _, v := range versions {
		server.RegisterTag(tagPrefix + v)
	}

	versions, err = mockPlugin.ListAll(t.Context())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	g := goldie.New(t)
	g.Assert(t, prefix+"_list_all", []byte(strings.Join(versions, "\n")))
}

// RunLatestStableGoldie runs the LatestStable goldie test with filter patterns.
func RunLatestStableGoldie(t *testing.T, cfg *PluginTestConfig) {
	t.Helper()

	testdataPath := cfg.TestdataPath
	prefix := goldenPrefix(cfg)
	listAllFile := prefix + "_list_all.golden"
	latestStableFile := prefix + "_latest_stable.golden"

	if isUpdateMode() {
		p := cfg.NewPlugin()

		version, err := p.LatestStable(t.Context(), "")
		if err != nil {
			t.Fatalf("LatestStable(\"\") failed: %v", err)
		}

		g := goldie.New(t)
		g.Assert(t, prefix+"_latest_stable", []byte(version))

		return
	}

	server := mock.NewServer(cfg.Config.RepoOwner, cfg.Config.RepoName)
	defer server.Close()

	githubClient := github.NewClientWithHTTP(server.Client(), server.URL())
	mockPlugin := cfg.NewPluginWithClient(githubClient)

	tagPrefix := cfg.Config.VersionPrefix
	if tagPrefix == "" {
		tagPrefix = "v"
	}

	if !GoldieFileExists(testdataPath, listAllFile) {
		t.Skip(listAllFile + " not found - run with ONLINE=1 to create")
	}

	versions, err := ReadGoldieVersions(testdataPath, listAllFile)
	if err != nil {
		t.Fatalf("failed to read goldie versions: %v", err)
	}

	for _, v := range versions {
		server.RegisterTag(tagPrefix + v)
	}

	// Derive filter pattern from latest stable (when available)
	var filterPattern string
	if GoldieFileExists(testdataPath, latestStableFile) {
		latest, err := ReadGoldieLatest(testdataPath, latestStableFile)
		if err != nil {
			t.Fatalf("failed to read goldie latest: %v", err)
		}

		parts := strings.Split(latest, ".")
		if len(parts) >= 2 {
			filterPattern = parts[0] + "." + parts[1]
		} else {
			filterPattern = latest
		}
	}

	tests := []struct {
		query string
		name  string
	}{
		{"", prefix + "_latest_stable"},
	}

	if filterPattern != "" {
		tests = append(tests, struct {
			query string
			name  string
		}{filterPattern, prefix + "_latest_stable_" + strings.ReplaceAll(filterPattern, ".", "_")})
	}

	goldieRecorder := goldie.New(t)
	for i := range tests {
		if !GoldieFileExists(testdataPath, tests[i].name+".golden") {
			t.Logf("Skipping %s - goldie file not found", tests[i].name)
			continue
		}

		version, err := mockPlugin.LatestStable(t.Context(), tests[i].query)
		if err != nil {
			t.Fatalf("LatestStable(%q) failed: %v", tests[i].query, err)
		}

		goldieRecorder.Assert(t, tests[i].name, []byte(version))
	}
}
