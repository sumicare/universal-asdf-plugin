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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ginkgo fixtures", func() {
	It("creates a fixture with a plugin", func() {
		cfg := newDummyPluginConfig("ginkgo-fixture-plugin")
		cfg.TestdataPath = GoldieTestDataPath(GinkgoT())

		fixture := newFixture(cfg)
		Expect(fixture).NotTo(BeNil())
		Expect(fixture.Plugin).NotTo(BeNil())

		fixture.Close()
	})

	It("creates a mock fixture with a mock server", func() {
		cfg := newDummyPluginConfig("ginkgo-mock-fixture-plugin")
		cfg.TestdataPath = GoldieTestDataPath(GinkgoT())

		fixture := newMockFixture(cfg)
		Expect(fixture).NotTo(BeNil())
		Expect(fixture.Plugin).NotTo(BeNil())
		Expect(fixture.Server).NotTo(BeNil())

		fixture.Close()
	})
})
