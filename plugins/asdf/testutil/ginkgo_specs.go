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

//nolint:revive // this file aggregates reusable Ginkgo specs; comment density and early-return rules are relaxed here
package testutil

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// newFixture creates a new test fixture from the config.
// In ONLINE mode, creates a real plugin; in mock mode, creates a mock plugin.
func newFixture(cfg *PluginTestConfig) *BinaryPluginTestFixture {
	if cfg.ForceMock {
		return NewBinaryPluginTestFixtureWithMode(cfg, true)
	}

	return NewBinaryPluginTestFixture(cfg, 3)
}

// newMockFixture creates a mock test fixture even in ONLINE mode.
// Use this for mock-specific tests that should always run with mock infrastructure.
func newMockFixture(cfg *PluginTestConfig) *BinaryPluginTestFixture {
	return NewBinaryPluginTestFixtureWithMode(cfg, true)
}

// DescribeBasicPluginBehavior generates Ginkgo specs for basic plugin behavior.
// This includes New, NewWithClient, Name, ListBinPaths, ExecEnv, ListLegacyFilenames, and Uninstall.
func DescribeBasicPluginBehavior(cfg *PluginTestConfig) {
	Describe("New", func() {
		It("creates a new plugin instance", func() {
			plugin := cfg.NewPlugin()
			Expect(plugin).NotTo(BeNil())
			Expect(plugin.Name()).To(Equal(cfg.Config.Name))
		})
	})

	Describe("NewWithClient", func() {
		It("creates a plugin with custom client", func() {
			fixture := newFixture(cfg)
			defer fixture.Close()

			Expect(fixture.Plugin).NotTo(BeNil())
		})
	})

	Describe("Name", func() {
		It("returns the plugin name", func() {
			plugin := cfg.NewPlugin()
			Expect(plugin.Name()).To(Equal(cfg.Config.Name))
		})
	})

	Describe("ListBinPaths", func() {
		It("returns bin paths", func() {
			plugin := cfg.NewPlugin()
			paths := plugin.ListBinPaths()
			Expect(paths).To(Equal("bin"))
		})
	})

	Describe("ExecEnv", func() {
		It("returns environment", func() {
			plugin := cfg.NewPlugin()
			env := plugin.ExecEnv("/tmp/install")

			Expect(env).To(BeEmpty())
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns legacy filenames", func() {
			plugin := cfg.NewPlugin()
			files := plugin.ListLegacyFilenames()

			Expect(files).To(BeEmpty())
		})
	})

	Describe("Uninstall", func() {
		It("removes installation directory", func() {
			plugin := cfg.NewPlugin()
			tempDir, err := os.MkdirTemp("", cfg.Config.Name+"-uninstall-*")
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Uninstall(context.Background(), tempDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tempDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
}

// DescribeListAll generates Ginkgo specs for ListAll behavior.
func DescribeListAll(cfg *PluginTestConfig) {
	Describe("ListAll", func() {
		var fixture *BinaryPluginTestFixture

		BeforeEach(func() {
			fixture = newFixture(cfg)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("lists versions", func() {
			if !asdf.IsOnline() {
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}

				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())
			}

			versions, err := fixture.Plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			if !asdf.IsOnline() {
				goldieVersions, err := fixture.GoldieVersions()
				Expect(err).NotTo(HaveOccurred())
				Expect(versions).To(ContainElements(goldieVersions))
			} else {
				Expect(len(versions)).To(BeNumerically(">", 1))
			}
		})
	})
}

// DescribeLatestStable generates Ginkgo specs for LatestStable behavior.
func DescribeLatestStable(cfg *PluginTestConfig) {
	Describe("LatestStable", func() {
		var fixture *BinaryPluginTestFixture

		BeforeEach(func() {
			fixture = newFixture(cfg)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns latest stable version", func() {
			if !asdf.IsOnline() { //nolint:revive // keep nesting for clarity in this test
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}

				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())
			}

			version, err := fixture.Plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())

			if !asdf.IsOnline() {
				goldieLatest, err := fixture.GoldieLatest()
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(goldieLatest))
			}
		})

		It("filters by pattern", func() {
			if !asdf.IsOnline() {
				if !fixture.GoldieFilesExist() {
					Skip("goldie files not found - run with ONLINE=1 to create")
				}

				Expect(fixture.SetupTagsFromGoldie()).To(Succeed())
			}

			filterPattern, err := fixture.GoldieFilterPattern()
			Expect(err).NotTo(HaveOccurred())

			version, err := fixture.Plugin.LatestStable(context.Background(), filterPattern)
			Expect(err).NotTo(HaveOccurred())

			if !asdf.IsOnline() {
				Expect(version).To(HavePrefix(filterPattern))
			} else {
				Expect(version).To(SatisfyAny(
					MatchRegexp(`^`+filterPattern+`\.\d+`),
					Equal(filterPattern),
				))
			}
		})
	})
}

// DescribeDownload generates Ginkgo specs for Download behavior.
func DescribeDownload(cfg *PluginTestConfig) {
	Describe("Download", func() {
		var fixture *BinaryPluginTestFixture

		BeforeEach(func() {
			fixture = newFixture(cfg)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("downloads binary", func() {
			platform, err := asdf.GetPlatform()
			Expect(err).NotTo(HaveOccurred())

			arch, err := asdf.GetArch()
			Expect(err).NotTo(HaveOccurred())

			testVersion, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			if !asdf.IsOnline() {
				fixture.SetupVersion(testVersion, platform, arch)
			}

			downloadDir, err := os.MkdirTemp("", cfg.Config.Name+"-download-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(downloadDir)

			err = fixture.Plugin.Download(context.Background(), testVersion, downloadDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})
}

// DescribeInstall generates Ginkgo specs for Install behavior.
func DescribeInstall(cfg *PluginTestConfig) {
	Describe("Install", func() {
		var fixture *BinaryPluginTestFixture

		BeforeEach(func() {
			fixture = newFixture(cfg)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("installs from downloaded archive", func() {
			platform, err := asdf.GetPlatform()
			Expect(err).NotTo(HaveOccurred())

			arch, err := asdf.GetArch()
			Expect(err).NotTo(HaveOccurred())

			testVersion, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			if !asdf.IsOnline() {
				fixture.SetupVersion(testVersion, platform, arch)
			}

			tempDir, err := os.MkdirTemp("", cfg.Config.Name+"-install-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")

			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			err = fixture.Plugin.Download(context.Background(), testVersion, downloadPath)
			Expect(err).NotTo(HaveOccurred())

			err = fixture.Plugin.Install(context.Background(), testVersion, downloadPath, installPath)
			Expect(err).NotTo(HaveOccurred())

			binaryPath := filepath.Join(installPath, "bin", cfg.Config.BinaryName)
			info, err := os.Stat(binaryPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()&asdf.ExecutablePermissionMask).NotTo(BeZero(), "binary should be executable")
		})
	})
}

// DescribeDownloadErrors generates Ginkgo specs for Download error cases.
func DescribeDownloadErrors(cfg *PluginTestConfig) {
	Describe("Download error cases [mock]", func() {
		var fixture *BinaryPluginTestFixture

		BeforeEach(func() {
			fixture = newMockFixture(cfg)
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("returns error for unsupported architecture", func() {
			downloadDir, err := os.MkdirTemp("", cfg.Config.Name+"-download-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(downloadDir)

			originalArch := os.Getenv("ASDF_OVERWRITE_ARCH")
			defer func() {
				if originalArch == "" {
					os.Unsetenv("ASDF_OVERWRITE_ARCH")
				} else {
					os.Setenv("ASDF_OVERWRITE_ARCH", originalArch)
				}
			}()

			testVersion, vErr := fixture.GoldieLatest()
			Expect(vErr).NotTo(HaveOccurred())

			os.Setenv("ASDF_OVERWRITE_ARCH", "unsupported")

			err = fixture.Plugin.Download(context.Background(), testVersion, downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported"))
		})

		It("returns error for non-existent version", func() {
			baseVersion, vErr := fixture.GoldieLatest()
			Expect(vErr).NotTo(HaveOccurred())

			nonExistentVersion := baseVersion + "-nonexistent"

			downloadDir, err := os.MkdirTemp("", cfg.Config.Name+"-download-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(downloadDir)

			err = fixture.Plugin.Download(context.Background(), nonExistentVersion, downloadDir)
			Expect(err).To(HaveOccurred())
		})
	})
}

// DescribeInstallErrors generates Ginkgo specs for Install error cases.
func DescribeInstallErrors(cfg *PluginTestConfig) {
	Describe("Install error cases [mock]", func() {
		It("returns error when archive doesn't exist", func() {
			plugin := cfg.NewPlugin()
			tempDir, err := os.MkdirTemp("", cfg.Config.Name+"-install-error-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")

			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			const testVersion = "test-version"

			err = plugin.Install(context.Background(), testVersion, downloadPath, installPath)
			Expect(err).To(HaveOccurred())
		})
	})
}

// DescribeMockOnlyListAll generates Ginkgo specs for mock-only ListAll tests.
func DescribeMockOnlyListAll(cfg *PluginTestConfig) {
	Describe("ListAll [mock]", func() {
		It("lists available versions", func() {
			fixture := newMockFixture(cfg)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			versions, err := fixture.Plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).NotTo(BeEmpty())

			goldieVersions, err := fixture.GoldieVersions()
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements(goldieVersions))
		})
	})
}

// DescribeMockOnlyLatestStable generates Ginkgo specs for mock-only LatestStable tests.
func DescribeMockOnlyLatestStable(cfg *PluginTestConfig) {
	Describe("LatestStable [mock]", func() {
		It("returns latest stable version", func() {
			fixture := newMockFixture(cfg)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			Expect(fixture.SetupTagsFromGoldie()).To(Succeed())

			version, err := fixture.Plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())

			goldieLatest, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(goldieLatest))
		})
	})
}

// DescribeMockOnlyDownload generates Ginkgo specs for mock-only Download tests.
func DescribeMockOnlyDownload(cfg *PluginTestConfig) {
	Describe("Download [mock]", func() {
		It("downloads binary", func() {
			fixture := newMockFixture(cfg)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			version, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			platform, pErr := asdf.GetPlatform()
			Expect(pErr).NotTo(HaveOccurred())

			arch, aErr := asdf.GetArch()
			Expect(aErr).NotTo(HaveOccurred())

			fixture.SetupVersion(version, platform, arch)

			downloadDir, err := os.MkdirTemp("", cfg.Config.Name+"-download-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(downloadDir)

			err = fixture.Plugin.Download(context.Background(), version, downloadDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})
}

// DescribeMockOnlyInstall generates Ginkgo specs for mock-only Install tests.
func DescribeMockOnlyInstall(cfg *PluginTestConfig) {
	Describe("Install [mock]", func() {
		It("installs from downloaded binary", func() {
			fixture := newMockFixture(cfg)
			defer fixture.Close()

			if !fixture.GoldieFilesExist() {
				Skip("goldie files not found - run with ONLINE=1 to create")
			}

			version, err := fixture.GoldieLatest()
			Expect(err).NotTo(HaveOccurred())

			platform, pErr := asdf.GetPlatform()
			Expect(pErr).NotTo(HaveOccurred())

			arch, aErr := asdf.GetArch()
			Expect(aErr).NotTo(HaveOccurred())

			fixture.SetupVersion(version, platform, arch)

			tempDir, err := os.MkdirTemp("", cfg.Config.Name+"-install-*")
			Expect(err).NotTo(HaveOccurred())

			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")

			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			err = fixture.Plugin.Download(context.Background(), version, downloadPath)
			Expect(err).NotTo(HaveOccurred())

			err = fixture.Plugin.Install(context.Background(), version, downloadPath, installPath)
			Expect(err).NotTo(HaveOccurred())

			binaryPath := filepath.Join(installPath, "bin", cfg.Config.BinaryName)

			_, err = os.Stat(binaryPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})
}
