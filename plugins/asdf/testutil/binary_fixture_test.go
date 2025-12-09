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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BinaryPluginTestFixture", func() {
	It("exposes goldie helpers and tracks files", func() {
		testdataDir, cleanup := CreateTestDir(GinkgoT())
		defer cleanup()

		cfg := newDummyPluginConfig("dummy-plugin")
		cfg.TestdataPath = testdataDir

		fixture := NewBinaryPluginTestFixtureWithMode(cfg, true)
		Expect(fixture.Server).NotTo(BeNil())
		defer fixture.Close()

		Expect(fixture.GoldenPrefix()).To(Equal("dummy_plugin"))
		Expect(fixture.ListAllGoldenFile()).To(Equal("dummy_plugin_list_all.golden"))
		Expect(fixture.LatestStableGoldenFile()).To(Equal("dummy_plugin_latest_stable.golden"))
		Expect(fixture.GoldieFilesExist()).To(BeFalse())

		Expect(os.WriteFile(filepath.Join(testdataDir, fixture.ListAllGoldenFile()), []byte("1.0.0\n1.1.0"), CommonFilePermission)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(testdataDir, fixture.LatestStableGoldenFile()), []byte("1.1.0"), CommonFilePermission)).To(Succeed())

		Expect(fixture.GoldieFilesExist()).To(BeTrue())

		versions, err := fixture.GoldieVersions()
		Expect(err).NotTo(HaveOccurred())
		Expect(versions).To(Equal([]string{"1.0.0", "1.1.0"}))

		latest, err := fixture.GoldieLatest()
		Expect(err).NotTo(HaveOccurred())
		Expect(latest).To(Equal("1.1.0"))

		pattern, err := fixture.GoldieFilterPattern()
		Expect(err).NotTo(HaveOccurred())
		Expect(pattern).NotTo(BeEmpty())
	})

	DescribeTable("registers downloads for all archive types", func(archiveType string) {
		cfg := newDummyPluginConfig("dummy-" + archiveType)
		cfg.Config.ArchiveType = archiveType

		fixture := NewBinaryPluginTestFixtureWithMode(cfg, true)
		Expect(fixture.Server).NotTo(BeNil(), "fixture.Server should not be nil for %s", archiveType)

		fixture.SetupVersion("1.0.0", "linux", "amd64")
		fixture.Close()
	},
		Entry("plain binary", ""),
		Entry("gz archive", "gz"),
		Entry("tar.gz archive", "tar.gz"),
		Entry("tar.xz archive", "tar.xz"),
		Entry("zip archive", "zip"),
	)
})
