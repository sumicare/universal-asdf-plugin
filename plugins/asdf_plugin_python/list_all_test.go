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
	"context"
	"net/http"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
	githubmock "github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

// Helper to check if any version has a prefix.
func hasVersionPrefix(versions []string, prefix string) bool {
	for _, v := range versions {
		if strings.HasPrefix(v, prefix) {
			return true
		}
	}

	return false
}

var _ = Describe("ListAll", func() {
	Describe("helper functions", func() {
		It("sortPythonVersions sorts correctly", func() {
			versions := []string{"3.9.0", "3.11.0", "3.10.0", "3.12.0"}
			sortPythonVersions(versions)
			Expect(versions).To(Equal([]string{"3.9.0", "3.10.0", "3.11.0", "3.12.0"}))
		})

		It("pythonBuildPath returns correct path", func() {
			Expect(New().pythonBuildPath()).To(ContainSubstring("python-build"))
		})
	})

	Describe("error cases [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns error when no versions found", func() {
			_, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
		})

		It("filters out alpha/beta/rc versions for stable", func() {
			fixture.SetupVersion("3.11.0")
			fixture.SetupVersion("3.12.0a1")
			fixture.SetupVersion("3.12.0b1")
			fixture.SetupVersion("3.12.0rc1")

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("3.11.0"))
		})
	})

	Describe("ListAllFromGitHub errors", func() {
		It("returns error on GitHub failure", func() {
			githubServer := githubmock.NewServer()
			defer githubServer.Close()
			githubClient := github.NewClientWithHTTP(&http.Client{}, githubServer.URL())

			mockPlugin := NewWithURLs("http://ftp.example.com", githubClient)

			_, err := mockPlugin.ListAllFromGitHub(context.Background())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ListAllFromGitHub [mock]", func() {
		It("lists versions from mocked GitHub tags", func() {
			server := githubmock.NewServer()
			defer server.Close()

			server.AddTags("python", "cpython", []string{"v3.11.0", "v3.11.1", "v3.12.0"})

			client := github.NewClientWithHTTP(server.HTTPServer.Client(), server.URL())
			plugin := NewWithURLs("https://mock.python.org/ftp/python/", client)

			versions, err := plugin.ListAllFromGitHub(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("3.11.0", "3.11.1", "3.12.0"))
		})
	})

	Describe("LatestStable edge cases [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns latest version with filter", func() {
			fixture.SetupVersion("3.10.0")
			fixture.SetupVersion("3.10.5")
			fixture.SetupVersion("3.11.0")

			version, err := fixture.plugin.LatestStable(context.Background(), "3.10")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("3.10.5"))
		})

		It("returns latest when no filter matches but versions exist", func() {
			fixture.SetupVersion("3.11.0")
			fixture.SetupVersion("3.12.0")

			version, err := fixture.plugin.LatestStable(context.Background(), "3.10")
			Expect(err).NotTo(HaveOccurred())

			Expect(version).To(Equal("3.12.0"))
		})

		It("returns error when only prereleases exist (FTP regex filters them)", func() {
			fixture.SetupVersion("3.13.0a1")
			fixture.SetupVersion("3.13.0b1")

			_, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errPythonNoVersionsFound))
		})
	})

	When("running online tests", func() {
		BeforeEach(func() {
			if !asdf.IsOnline() {
				Skip("skipping online test (set ONLINE=1 to run)")
			}
		})

		It("lists versions from real FTP", func() {
			versions, err := New().ListAllFromFTP(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
			Expect(hasVersionPrefix(versions, "3.11")).To(BeTrue())
		})

		It("lists versions from GitHub", func() {
			versions, err := New().ListAllFromGitHub(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
			Expect(hasVersionPrefix(versions, "3.11")).To(BeTrue())
		})
	})

	When("testing python-build integration", func() {
		BeforeEach(func() {
			if !asdf.IsOnline() {
				Skip("skipping online test (set ONLINE=1 to run)")
			}
		})

		It("installs python-build on first call", func() {
			plugin := NewWithBuildDir(testutil.TestBuildDir(GinkgoT(), "pyenv"))

			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
			Expect(hasVersionPrefix(versions, "3.11") || hasVersionPrefix(versions, "3.12")).To(BeTrue())
		})

		It("updates python-build on subsequent calls", func() {
			plugin := NewWithBuildDir(testutil.TestBuildDir(GinkgoT(), "pyenv"))

			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
		})
	})

	Describe("ListAll with mock python-build [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
		})

		It("lists versions using python-build", func() {
			versions, err := fixture.plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			Expect(versions).To(ContainElement("3.11.0"))
		})

		It("ensures python-build is available", func() {
			err := fixture.plugin.ensurePythonBuild(context.Background())
			Expect(err).NotTo(HaveOccurred())

			pythonBuildPath := fixture.plugin.pythonBuildPath()
			_, err = os.Stat(pythonBuildPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("calls ensurePythonBuild multiple times", func() {
			err := fixture.plugin.ensurePythonBuild(context.Background())
			Expect(err).NotTo(HaveOccurred())

			err = fixture.plugin.ensurePythonBuild(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns pythonBuildPath correctly", func() {
			path := fixture.plugin.pythonBuildPath()
			Expect(path).To(ContainSubstring("python-build"))
			Expect(path).To(ContainSubstring("bin"))
		})
	})

	Describe("goldie helpers", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
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

		It("sets up tags from goldie for ListAllFromFTP", func() {
			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			versions, err := fixture.plugin.ListAllFromFTP(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())
		})
	})
})
