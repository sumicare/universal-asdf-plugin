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
	"testing"

	. "github.com/onsi/gomega"
)

// TestGoldieHelpers verifies the basic behavior of the goldie helper methods
// on pythonTestFixture. These tests are mock-only and exercise the thin
// wrappers around testutil helpers so they contribute to coverage.
func TestGoldieHelpers(t *testing.T) {
	RegisterTestingT(t)

	fixture := newPythonTestFixtureWithMode(true)
	defer fixture.Close()

	if !fixture.GoldieFilesExist() {
		t.Skip("goldie files not found - run with ONLINE=1 to create")
	}

	Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

	versions, err := fixture.GoldieVersions()
	Expect(err).NotTo(HaveOccurred())
	Expect(versions).NotTo(BeEmpty())

	latest, err := fixture.GoldieLatest()
	Expect(err).NotTo(HaveOccurred())
	Expect(latest).NotTo(BeEmpty())

	pattern, err := fixture.GoldieFilterPattern()
	Expect(err).NotTo(HaveOccurred())
	Expect(pattern).NotTo(BeEmpty())
}
