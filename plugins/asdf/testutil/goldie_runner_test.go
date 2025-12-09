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

import "testing"

// TestDummyPluginGoldieRunners exercises the Goldie runner helpers with the dummy plugin.
func TestDummyPluginGoldieRunners(t *testing.T) {
	if isUpdateMode() {
		t.Skip("skipping dummy plugin goldie runners in update mode")
	}

	cfg := newDummyPluginConfig("dummy-plugin")

	cfg.TestdataPath = GoldieTestDataPath(t)

	RunHelpGoldie(t, cfg)
	RunListBinPathsGoldie(t, cfg)
	RunListLegacyFilenamesGoldie(t, cfg)
	RunExecEnvGoldie(t, cfg)
	RunListAllGoldie(t, cfg)
	RunLatestStableGoldie(t, cfg)
}
