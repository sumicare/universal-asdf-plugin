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

package asdf_plugin_cosign

import (
	"testing"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// TestHelpGoldie verifies that the plugin help output matches the golden snapshot.
func TestHelpGoldie(t *testing.T) {
	testutil.RunHelpGoldie(t, pluginTestConfig())
}

// TestListBinPathsGoldie verifies that ListBinPaths output matches the golden snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	testutil.RunListBinPathsGoldie(t, pluginTestConfig())
}

// TestListLegacyFilenamesGoldie verifies that ListLegacyFilenames output matches the golden snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	testutil.RunListLegacyFilenamesGoldie(t, pluginTestConfig())
}

// TestExecEnvGoldie verifies that ExecEnv output matches the golden snapshot.
func TestExecEnvGoldie(t *testing.T) {
	testutil.RunExecEnvGoldie(t, pluginTestConfig())
}

// TestListAllGoldie verifies that ListAll output matches the golden snapshot.
func TestListAllGoldie(t *testing.T) {
	testutil.RunListAllGoldie(t, pluginTestConfig())
}

// TestLatestStableGoldie verifies that LatestStable behavior matches the golden snapshots.
func TestLatestStableGoldie(t *testing.T) {
	testutil.RunLatestStableGoldie(t, pluginTestConfig())
}
