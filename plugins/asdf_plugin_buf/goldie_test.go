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

package asdf_plugin_buf

import (
	"testing"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// TestHelpGoldie verifies that the buf plugin help output matches the goldie snapshot.
func TestHelpGoldie(t *testing.T) {
	testutil.RunHelpGoldie(t, pluginTestConfig())
}

// TestListBinPathsGoldie verifies that the buf plugin bin paths match the goldie snapshot.
func TestListBinPathsGoldie(t *testing.T) {
	testutil.RunListBinPathsGoldie(t, pluginTestConfig())
}

// TestListLegacyFilenamesGoldie verifies that the buf plugin legacy filenames match the goldie snapshot.
func TestListLegacyFilenamesGoldie(t *testing.T) {
	testutil.RunListLegacyFilenamesGoldie(t, pluginTestConfig())
}

// TestExecEnvGoldie verifies that the buf plugin exec environment matches the goldie snapshot.
func TestExecEnvGoldie(t *testing.T) {
	testutil.RunExecEnvGoldie(t, pluginTestConfig())
}

// TestListAllGoldie verifies that the buf plugin ListAll output matches the goldie snapshot.
func TestListAllGoldie(t *testing.T) {
	testutil.RunListAllGoldie(t, pluginTestConfig())
}

// TestLatestStableGoldie verifies that the buf plugin LatestStable output matches the goldie snapshots.
func TestLatestStableGoldie(t *testing.T) {
	testutil.RunLatestStableGoldie(t, pluginTestConfig())
}
