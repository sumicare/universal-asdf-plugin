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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

var _ = Describe("Install", func() {
	Describe("installDefaultPackages", func() {
		var plugin *Plugin

		BeforeEach(func() {
			plugin = New()
		})

		It("returns nil when no default packages file exists", func() {
			tempDir, err := os.MkdirTemp("", "node-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Unsetenv("ASDF_NPM_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), tempDir)).To(Succeed())
		})

		It("returns nil when custom file does not exist", func() {
			tempDir, err := os.MkdirTemp("", "node-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Setenv("ASDF_NPM_DEFAULT_PACKAGES_FILE", "/nonexistent/file")
			defer os.Unsetenv("ASDF_NPM_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), tempDir)).To(Succeed())
		})

		It("reads packages from file with comments", func() {
			tempDir, err := os.MkdirTemp("", "node-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			pkgFile := filepath.Join(tempDir, "pkgs")
			err = os.WriteFile(pkgFile, []byte("# comment\n\ntypescript\neslint # inline\n"), testutil.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("ASDF_NPM_DEFAULT_PACKAGES_FILE", pkgFile)
			defer os.Unsetenv("ASDF_NPM_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), tempDir)).To(Succeed())
		})
	})

	Describe("enableCorepack", func() {
		var plugin *Plugin

		BeforeEach(func() {
			plugin = New()
		})

		It("returns nil when corepack not found", func() {
			tempDir, err := os.MkdirTemp("", "node-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			Expect(plugin.enableCorepack(context.Background(), tempDir)).To(Succeed())
		})

		It("runs corepack when binary exists", func() {
			if testing.Short() {
				Skip("skipping corepack exec test in short mode")
			}

			installDir, err := os.MkdirTemp("", "node-corepack-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			binDir := filepath.Join(installDir, "bin")
			Expect(os.MkdirAll(binDir, asdf.CommonDirectoryPermission)).To(Succeed())

			corepackPath := filepath.Join(binDir, "corepack")

			script := []byte("#!/bin/sh\nexit 0\n")
			Expect(os.WriteFile(corepackPath, script, asdf.CommonDirectoryPermission)).To(Succeed())

			Expect(plugin.enableCorepack(context.Background(), installDir)).To(Succeed())
		})
	})

	{
		Describe("Install", func() {
			var originalInstallPythonToolchain func(context.Context) error

			BeforeEach(func() {
				originalInstallPythonToolchain = installPythonToolchain
				installPythonToolchain = func(_ context.Context) error {
					return nil
				}
			})

			AfterEach(func() {
				installPythonToolchain = originalInstallPythonToolchain
			})
			var fixture *nodeTestFixture

			BeforeEach(func() {
				fixture = newNodeMockFixture()
			})

			AfterEach(func() {
				if fixture != nil {
					fixture.Close()
				}
			})

			It("downloads and installs Node.js", func() {
				fixture.SetupVersion("20.10.0", "linux", "x64", false)

				downloadDir, err := os.MkdirTemp("", "node-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)
				installDir, err := os.MkdirTemp("", "node-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				err = fixture.plugin.Download(context.Background(), "20.10.0", downloadDir)
				Expect(err).NotTo(HaveOccurred())

				err = fixture.plugin.Install(context.Background(), "20.10.0", downloadDir, installDir)
				Expect(err).NotTo(HaveOccurred())

				nodeBin := filepath.Join(installDir, "bin", "node")
				_, err = os.Stat(nodeBin)
				Expect(err).NotTo(HaveOccurred())

				if !asdf.IsOnline() {
					return
				}

				out, err := exec.Command(nodeBin, "--version").Output()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(out)).To(HavePrefix("v20.10.0"))
			})

			It("auto-downloads if archive missing", func() {
				fixture.SetupVersion("20.10.0", "linux", "x64", false)

				downloadDir, err := os.MkdirTemp("", "node-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)
				installDir, err := os.MkdirTemp("", "node-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				err = fixture.plugin.Install(context.Background(), "20.10.0", downloadDir, installDir)
				Expect(err).NotTo(HaveOccurred())

				_, err = os.Stat(filepath.Join(installDir, "bin", "node"))
				Expect(err).NotTo(HaveOccurred())
			})

			if !asdf.IsOnline() {
				It("installs with corepack enabled (mock)", func() {
					fixture.SetupVersion("20.10.0", "linux", "x64", false)

					downloadDir, err := os.MkdirTemp("", "node-*")
					Expect(err).NotTo(HaveOccurred())
					defer os.RemoveAll(downloadDir)
					installDir, err := os.MkdirTemp("", "node-*")
					Expect(err).NotTo(HaveOccurred())
					defer os.RemoveAll(installDir)

					os.Setenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK", "1")
					defer os.Unsetenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK")

					Expect(fixture.plugin.Download(context.Background(), "20.10.0", downloadDir)).To(Succeed())

					Expect(fixture.plugin.Install(context.Background(), "20.10.0", downloadDir, installDir)).To(Succeed())

					_, err = os.Stat(filepath.Join(installDir, "bin", "node"))
					Expect(err).NotTo(HaveOccurred())
				})
			} else {
				It("installs default npm packages using installed Node.js", func() {
					if !asdf.IsOnline() {
						return
					}

					fixture.SetupVersion("20.10.0", "linux", "x64", false)

					downloadDir := testutil.TestDownloadDir(GinkgoT(), "nodejs-20.10.0")
					installPath := testutil.TestInstallDir(GinkgoT(), "nodejs-20.10.0")

					if _, err := os.Stat(filepath.Join(installPath, "bin", "node")); os.IsNotExist(err) {
						os.RemoveAll(installPath)
						Expect(fixture.plugin.Download(context.Background(), "20.10.0", downloadDir)).To(Succeed())
						Expect(fixture.plugin.Install(context.Background(), "20.10.0", downloadDir, installPath)).To(Succeed())
					}

					tmpDir, cleanup := testutil.CreateTestDir(GinkgoT())
					defer cleanup()

					pkgFile := filepath.Join(tmpDir, "default-npm-packages")
					Expect(os.WriteFile(pkgFile, []byte("# Test packages\nnpm-check\n"), testutil.CommonFilePermission)).To(Succeed())

					os.Setenv("ASDF_NPM_DEFAULT_PACKAGES_FILE", pkgFile)
					defer os.Unsetenv("ASDF_NPM_DEFAULT_PACKAGES_FILE")

					err := fixture.plugin.installDefaultPackages(context.Background(), installPath)
					Expect(err).NotTo(HaveOccurred())
				})
			}
		})
	}
})
