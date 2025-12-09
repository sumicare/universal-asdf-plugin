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
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ListAll", func() {
	Describe("parseGoVersions", func() {
		DescribeTable("extracts versions from git output",
			func(input string, expected []string) {
				versions := parseGoVersions(input)
				Expect(versions).To(Equal(expected))
			},
			Entry("standard versions",
				"abc123\trefs/tags/go1.20.0\ndef456\trefs/tags/go1.21.0",
				[]string{"1.20.0", "1.21.0"}),
			Entry("with rc versions",
				"abc123\trefs/tags/go1.21rc1\ndef456\trefs/tags/go1.21.0",
				[]string{"1.21rc1", "1.21.0"}),
			Entry("with peeled refs",
				"abc123\trefs/tags/go1.21.0\ndef456\trefs/tags/go1.21.0^{}",
				[]string{"1.21.0"}),
		)
	})

	Describe("parseGoTags", func() {
		It("extracts versions from tag names", func() {
			tags := []string{"go1.20.0", "go1.21.0", "go1.21rc1", "release.r60"}
			versions := parseGoTags(tags)

			Expect(versions).To(HaveLen(3))
			Expect(versions).To(ContainElements("1.20.0", "1.21.0", "1.21rc1"))
		})
	})

	Describe("filterOldVersions", func() {
		It("filters out old Go versions", func() {
			versions := []string{"1", "1.0", "1.0.1", "1.1", "1.2.2", "1.3", "1.20.0"}
			filtered := filterOldVersions(versions)

			Expect(filtered).NotTo(ContainElement("1"))
			Expect(filtered).NotTo(ContainElement("1.0"))
			Expect(filtered).NotTo(ContainElement("1.0.1"))
			Expect(filtered).To(ContainElement("1.2.2"))
			Expect(filtered).To(ContainElement("1.3"))
			Expect(filtered).To(ContainElement("1.20.0"))
		})
	})

	Describe("sortGoVersions", func() {
		It("sorts versions correctly", func() {
			versions := []string{"1.3", "1.20.0", "1.2.2", "1.21.0"}
			sortGoVersions(versions)

			Expect(versions).To(Equal([]string{"1.2.2", "1.3", "1.20.0", "1.21.0"}))
		})
	})

	Describe("compareGoVersions", func() {
		DescribeTable("compares versions",
			func(a, b string, expectedSign int) {
				result := compareGoVersions(a, b)
				if expectedSign < 0 {
					Expect(result).To(BeNumerically("<", 0))
				} else if expectedSign > 0 {
					Expect(result).To(BeNumerically(">", 0))
				} else {
					Expect(result).To(Equal(0))
				}
			},
			Entry("1.20.0 < 1.21.0", "1.20.0", "1.21.0", -1),
			Entry("1.21.0 > 1.20.0", "1.21.0", "1.20.0", 1),
			Entry("1.20.0 == 1.20.0", "1.20.0", "1.20.0", 0),
			Entry("1.3 < 1.20", "1.3", "1.20", -1),
		)
	})

	{
		Describe("ListAll", func() {
			var fixture *goTestFixture

			BeforeEach(func() {
				fixture = newGoMockFixture()
			})

			AfterEach(func() {
				fixture.Close()
			})

			It("lists Go versions", func() {
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}

				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

				versions, err := fixture.plugin.ListAll(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(versions).NotTo(BeEmpty())

				goldieVersions, gErr := fixture.GoldieVersions()
				Expect(gErr).NotTo(HaveOccurred())
				expected := strings.TrimSpace(strings.Join(goldieVersions, "\n"))
				Expect(strings.Join(versions, "\n")).To(Equal(expected))
			})

			It("returns latest stable version", func() {
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}

				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

				version, err := fixture.plugin.LatestStable(context.Background(), "")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).NotTo(BeEmpty())

				goldieLatest, gErr := fixture.GoldieLatest()
				Expect(gErr).NotTo(HaveOccurred())
				Expect(version).To(Equal(goldieLatest))
			})

			It("returns latest stable version with filter", func() {
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}

				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

				filterPattern, err := fixture.GoldieFilterPattern()
				Expect(err).NotTo(HaveOccurred())
				Expect(filterPattern).NotTo(BeEmpty())

				version, err := fixture.plugin.LatestStable(context.Background(), filterPattern)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).NotTo(BeEmpty())
				Expect(version).To(HavePrefix(filterPattern))
			})
		})
	}

	Describe("goldie helpers", func() {
		var fixture *goTestFixture

		BeforeEach(func() {
			fixture = newGoMockFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("reads versions and latest from goldie", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			versions, err := fixture.GoldieVersions()
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			latest, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeEmpty())

			pattern, err := fixture.GoldieFilterPattern()
			Expect(err).NotTo(HaveOccurred())
			Expect(pattern).NotTo(BeEmpty())
		})

		It("sets up tags from goldie for ListAll", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			_, err := fixture.plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("reports whether goldie files exist", func() {
			_ = fixture.GoldieFilesExist()
		})
	})
})
