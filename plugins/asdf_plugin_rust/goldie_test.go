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

package asdf_plugin_rust

import (
	"sort"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
)

// TestHelpGoldie verifies that the plugin help output matches the golden snapshot.
func TestHelpGoldie(t *testing.T) {
	p := New()
	h := p.Help()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "rust_help", []byte(strings.Join([]string{h.Overview, h.Deps, h.Config, h.Links}, "\n\n")))
}

// TestListBinPathsGoldie verifies that ListBinPaths output matches the golden snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	p := New()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "rust_bin_paths", []byte(p.ListBinPaths()))
}

// TestListLegacyFilenamesGoldie verifies that ListLegacyFilenames output matches the golden snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	p := New()
	files := p.ListLegacyFilenames()
	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "rust_legacy_filenames", []byte(strings.Join(files, "\n")))
}

// TestExecEnvGoldie verifies that ExecEnv output matches the golden snapshot.
func TestExecEnvGoldie(t *testing.T) {
	p := New()
	env := p.ExecEnv("/tmp/install")

	output := make([]string, 0, len(env))
	for k, v := range env {
		output = append(output, k+"="+v)
	}

	sort.Strings(output)

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "rust_exec_env", []byte(strings.Join(output, "\n")))
}
