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

package asdf_plugin_nodejs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

var _ = Describe("ListAll", func() {
	Describe("helper functions", func() {
		It("getNodeArch returns valid architecture", func() {
			arch, err := getNodeArch()
			Expect(err).NotTo(HaveOccurred())
			Expect([]string{"x64", "x86", "arm64", "armv7l"}).To(ContainElement(arch))
		})

		Describe("listAllFromAPI [mock]", func() {
			It("lists versions from mocked index", func() {
				fixture := newNodeMockFixture()
				defer fixture.Close()

				fixture.SetupVersion("18.12.0", "linux", "x64", "Hydrogen")
				fixture.SetupVersion("20.9.0", "linux", "x64", "Iron")
				fixture.SetupVersion("21.0.0", "linux", "x64", false)

				versions, err := fixture.plugin.listAllFromAPI(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(versions).To(ContainElements("18.12.0", "20.9.0", "21.0.0"))
			})
		})

		It("nodeBuildPath returns correct path", func() {
			Expect(New().nodeBuildPath()).To(ContainSubstring("node-build"))
		})

		Describe("listAllFromNodeBuild [mock]", func() {
			var originalExecFn func(context.Context, string, ...string) *exec.Cmd

			BeforeEach(func() {
				originalExecFn = execCommandContextFnNode
			})

			AfterEach(func() {
				execCommandContextFnNode = originalExecFn
			})

			It("lists numeric versions and filters out non-numeric definitions", func() {
				plugin := NewWithBuildDir("/tmp/fake-node-build-dir")

				execCommandContextFnNode = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
					return exec.CommandContext(ctx, "bash", "-c", "printf '18.12.0\niojs-3.0.0\n20.10.0\n'")
				}

				versions, err := plugin.listAllFromNodeBuild(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(versions).To(Equal([]string{"18.12.0", "20.10.0"}))
			})

			It("returns error when node-build command fails", func() {
				plugin := NewWithBuildDir("/tmp/fake-node-build-dir")

				execCommandContextFnNode = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
					return exec.CommandContext(ctx, "bash", "-c", "exit 1")
				}

				versions, err := plugin.listAllFromNodeBuild(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(versions).To(BeNil())
			})
		})

		Describe("ensureNodeBuild [mock]", func() {
			var (
				originalExecFn func(context.Context, string, ...string) *exec.Cmd
				tempDir        string
			)

			BeforeEach(func() {
				originalExecFn = execCommandContextFnNode
				var err error
				tempDir, err = os.MkdirTemp("", "node-build-test-*")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				execCommandContextFnNode = originalExecFn
				os.RemoveAll(tempDir)
			})

			It("clones node-build when it is missing", func() {
				buildDir := filepath.Join(tempDir, "node-build-clone")
				plugin := NewWithBuildDir(buildDir)

				var capturedArgs []string
				execCommandContextFnNode = func(ctx context.Context, name string, args ...string) *exec.Cmd {
					capturedArgs = append([]string{name}, args...)

					return exec.CommandContext(ctx, "bash", "-c", "exit 0")
				}

				Expect(plugin.ensureNodeBuild(context.Background())).To(Succeed())
				Expect(capturedArgs).NotTo(BeEmpty())
				Expect(capturedArgs[0]).To(Equal("git"))
				Expect(capturedArgs).To(ContainElements("clone", nodeBuildGitURL))
			})

			It("updates node-build when it already exists", func() {
				buildDir := filepath.Join(tempDir, "node-build-update")
				binDir := filepath.Join(buildDir, "bin")
				Expect(os.MkdirAll(binDir, asdf.CommonDirectoryPermission)).To(Succeed())

				Expect(os.WriteFile(filepath.Join(binDir, "node-build"), []byte("#!/bin/sh\nexit 0\n"), asdf.CommonFilePermission)).To(Succeed())

				plugin := NewWithBuildDir(buildDir)

				var capturedArgs []string
				execCommandContextFnNode = func(ctx context.Context, name string, args ...string) *exec.Cmd {
					capturedArgs = append([]string{name}, args...)
					return exec.CommandContext(ctx, "bash", "-c", "exit 0")
				}

				Expect(plugin.ensureNodeBuild(context.Background())).To(Succeed())
				Expect(capturedArgs).NotTo(BeEmpty())
				Expect(capturedArgs[0]).To(Equal("git"))
				Expect(capturedArgs).To(ContainElements("-C", buildDir, "pull", "--ff-only"))
			})
		})
	})

	{
		Describe("ListAll", func() {
			var fixture *nodeTestFixture

			BeforeEach(func() {
				fixture = newNodeTestFixture()
			})

			AfterEach(func() {
				fixture.Close()
			})

			It("lists Node.js versions", func() {
				if !asdf.IsOnline() {

					fixture.SetupVersion("18.19.0", "linux", "x64", "Hydrogen")
					fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")
					fixture.SetupVersion("21.0.0", "linux", "x64", false)
				}

				versions, err := fixture.plugin.ListAll(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(versions).NotTo(BeEmpty())

				if !asdf.IsOnline() {
					Expect(versions).To(ContainElements("18.19.0", "20.10.0", "21.0.0"))
				} else {

					found := false
					for _, v := range versions {
						if strings.HasPrefix(v, "20.") {
							found = true
							break
						}
					}
					Expect(found).To(BeTrue(), "expected to find Node.js 20.x version")
				}
			})

			It("returns latest stable version", func() {
				if !asdf.IsOnline() {
					fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")
					fixture.SetupVersion("21.0.0", "linux", "x64", false)
				}

				version, err := fixture.plugin.LatestStable(context.Background(), "")
				Expect(err).NotTo(HaveOccurred())

				if !asdf.IsOnline() {
					Expect(version).To(Equal("20.10.0"))
				} else {
					Expect(version).NotTo(BeEmpty())
					Expect(version).To(MatchRegexp(`^\d+\.\d+\.\d+$`))
				}
			})

			It("resolves LTS aliases", func() {
				if !asdf.IsOnline() {
					fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")
				}

				for _, alias := range []string{"lts", "lts/*", "lts/iron"} {

					resolved, err := fixture.plugin.ResolveVersion(context.Background(), alias)
					Expect(err).NotTo(HaveOccurred())

					if !asdf.IsOnline() {
						Expect(resolved).To(Equal("20.10.0"))
					} else {
						Expect(resolved).NotTo(Equal(alias))
						Expect(resolved).To(MatchRegexp(`^\d+\.\d+\.\d+$`))
					}
				}
			})
		})
	}

	Describe("error cases [mock]", func() {
		var fixture *nodeTestFixture

		BeforeEach(func() {
			fixture = newNodeMockFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns error for unknown LTS codename", func() {
			fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")
			_, err := fixture.plugin.LatestStable(context.Background(), "lts/unknown")
			Expect(err).To(HaveOccurred())
		})

		It("returns error when no LTS versions available", func() {
			fixture.SetupVersion("21.0.0", "linux", "x64", false)
			_, err := fixture.plugin.getLatestLTS(context.Background())
			Expect(err).To(HaveOccurred())
		})

		It("prefers odd-major LTS versions over non-LTS odd versions", func() {
			fixture.SetupVersion("19.0.0", "linux", "x64", "LTS-19")
			fixture.SetupVersion("21.0.0", "linux", "x64", false)

			version, err := fixture.plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("19.0.0"))
		})

		It("falls back to first matching version when no LTS-eligible majors exist", func() {
			fixture.SetupVersion("19.0.0", "linux", "x64", false)
			fixture.SetupVersion("21.1.0", "linux", "x64", false)

			version, err := fixture.plugin.LatestStable(context.Background(), "19")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("19.0.0"))
		})

		It("returns error when no stable version matches the query prefix", func() {
			fixture.SetupVersion("18.12.0", "linux", "x64", "Hydrogen")
			fixture.SetupVersion("20.10.0", "linux", "x64", "Iron")

			_, err := fixture.plugin.LatestStable(context.Background(), "99")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no stable version found"))
		})
	})

	When("testing node-build integration [online]", func() {
		BeforeEach(func() {
			if !asdf.IsOnline() {
				Skip("skipping online test")
			}
		})

		It("installs and uses node-build", func() {
			plugin := NewWithBuildDir(testutil.TestBuildDir(GinkgoT(), "node-build"))

			for range 2 {
				versions, err := plugin.ListAll(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(versions).NotTo(BeEmpty())

				found := false
				for _, v := range versions {
					if strings.HasPrefix(v, "20.") {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			}
		})
	})
})
