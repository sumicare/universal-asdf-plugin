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
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var _ = Describe("Install", func() {
	Describe("installDefaultPackages", func() {
		var plugin *Plugin

		BeforeEach(func() {
			plugin = New()
		})

		It("returns nil when no default packages file exists", func() {
			tempDir, err := os.MkdirTemp("", "python-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), tempDir)).To(Succeed())
		})

		It("returns nil when custom file does not exist", func() {
			tempDir, err := os.MkdirTemp("", "python-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Setenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE", "/nonexistent/file")
			defer os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), tempDir)).To(Succeed())
		})

		It("reads packages from file with comments", func() {
			tempDir, err := os.MkdirTemp("", "python-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			binDir := filepath.Join(tempDir, "bin")
			Expect(os.MkdirAll(binDir, asdf.CommonDirectoryPermission)).To(Succeed())
			pipPath := filepath.Join(binDir, "pip")
			Expect(os.WriteFile(pipPath, []byte("#!/bin/sh\nexit 0\n"), asdf.CommonDirectoryPermission)).To(Succeed())

			pkgFile := filepath.Join(tempDir, "pkgs")
			Expect(os.WriteFile(pkgFile, []byte("# comment\n\npip\nsetuptools # inline\n"), asdf.CommonFilePermission)).To(Succeed())
			os.Setenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE", pkgFile)
			defer os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), tempDir)).To(Succeed())
		})
	})

	Describe("verifyBuildDeps", func() {
		var plugin *Plugin
		var originalExecFn func(context.Context, string, ...string) *exec.Cmd

		BeforeEach(func() {
			plugin = New()
			originalExecFn = execCommandContextFnPython
		})

		AfterEach(func() {
			execCommandContextFnPython = originalExecFn
			os.Unsetenv("ASDF_PYTHON_SKIP_SYSDEPS_CHECK")
		})

		It("skips checks when ASDF_PYTHON_SKIP_SYSDEPS_CHECK is set", func() {
			os.Setenv("ASDF_PYTHON_SKIP_SYSDEPS_CHECK", "1")
			Expect(plugin.verifyBuildDeps(context.Background())).To(Succeed())
		})

		It("returns nil when ldconfig is not available", func() {
			os.Unsetenv("ASDF_PYTHON_SKIP_SYSDEPS_CHECK")

			execCommandContextFnPython = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
				_ = args
				return exec.CommandContext(ctx, "false")
			}

			Expect(plugin.verifyBuildDeps(context.Background())).To(Succeed())
		})

		It("returns nil when all required libraries are present", func() {
			output := "libbz2.so\nlibreadline.so\nlibncursesw.so\nlibssl.so\nlibsqlite3.so\n" +
				"libgdbm.so\nlibffi.so\nlibz.so\nlibuuid.so\nliblzma.so\n"

			execCommandContextFnPython = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
				_ = args
				return exec.CommandContext(ctx, "printf", "%s", output)
			}

			Expect(plugin.verifyBuildDeps(context.Background())).To(Succeed())
		})

		It("returns error when required libraries are missing", func() {
			output := "libbz2.so\nlibreadline.so\n"

			execCommandContextFnPython = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
				_ = args
				return exec.CommandContext(ctx, "printf", "%s", output)
			}

			err := plugin.verifyBuildDeps(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("libssl.so"))
		})
	})

	Describe("ReadDefaultPackages", func() {
		var plugin *Plugin

		BeforeEach(func() {
			plugin = New()
		})

		It("returns packages from file", func() {
			tempDir, err := os.MkdirTemp("", "python-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			pkgFile := filepath.Join(tempDir, "pkgs")
			Expect(os.WriteFile(pkgFile, []byte("# comment\n\npip\nsetuptools\nwheel\n"), asdf.CommonFilePermission)).To(Succeed())
			os.Setenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE", pkgFile)
			defer os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			packages, err := plugin.ReadDefaultPackages()
			Expect(err).NotTo(HaveOccurred())
			Expect(packages).To(Equal([]string{"pip", "setuptools", "wheel"}))
		})

		It("returns nil when file does not exist", func() {
			os.Setenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE", "/nonexistent/packages")
			defer os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			packages, err := plugin.ReadDefaultPackages()
			Expect(err).NotTo(HaveOccurred())
			Expect(packages).To(BeNil())
		})

		It("returns nil when default file does not exist", func() {
			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			packages, err := plugin.ReadDefaultPackages()
			Expect(err).NotTo(HaveOccurred())
			Expect(packages).To(BeNil())
		})

		It("filters out comments and empty lines", func() {
			tempDir, err := os.MkdirTemp("", "python-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			pkgFile := filepath.Join(tempDir, "pkgs")
			content := "# This is a comment\n\n  \npip\n# Another comment\nsetuptools\n\n"
			Expect(os.WriteFile(pkgFile, []byte(content), asdf.CommonFilePermission)).To(Succeed())
			os.Setenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE", pkgFile)
			defer os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			packages, err := plugin.ReadDefaultPackages()
			Expect(err).NotTo(HaveOccurred())
			Expect(packages).To(Equal([]string{"pip", "setuptools"}))
		})
	})

	{
		Describe("Install [mock]", func() {
			var fixture *pythonTestFixture

			BeforeEach(func() {
				fixture = newPythonTestFixtureWithMode(true)
			})

			AfterEach(func() {
				if fixture != nil {
					fixture.Close()
				}
			})

			It("downloads and installs Python", func() {
				version := "3.11.0"
				fixture.SetupVersion(version)

				downloadDir, err := os.MkdirTemp("", "python-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)
				installDir, err := os.MkdirTemp("", "python-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

				if !asdf.IsOnline() {

					err := fixture.plugin.Install(context.Background(), version, downloadDir, installDir)
					Expect(err).NotTo(HaveOccurred())

					pythonBin := filepath.Join(installDir, "bin", "python")
					_, err = os.Stat(pythonBin)
					Expect(err).NotTo(HaveOccurred())
				} else {

					err := fixture.plugin.Install(context.Background(), version, downloadDir, installDir)
					Expect(err).NotTo(HaveOccurred())

					pythonBin := filepath.Join(installDir, "bin", "python")
					_, err = os.Stat(pythonBin)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})
	}

	Describe("Install with patches [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
		})

		It("installs with patch from URL", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			patchContent := "mock patch content"
			fixture.server.RegisterFile("/patches/test.patch", []byte(patchContent))

			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "python-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			os.Setenv("ASDF_PYTHON_PATCH_URL", fixture.server.URL()+"/patches/test.patch")
			defer os.Unsetenv("ASDF_PYTHON_PATCH_URL")
			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			err = fixture.plugin.Install(context.Background(), version, downloadDir, installDir)
			Expect(err).NotTo(HaveOccurred())

			pythonBin := filepath.Join(installDir, "bin", "python")
			_, err = os.Stat(pythonBin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("installs with patch from directory", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			patchDir, err := os.MkdirTemp("", "python-patches-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(patchDir)

			patchFile := filepath.Join(patchDir, version+".patch")
			Expect(os.WriteFile(patchFile, []byte("mock patch"), asdf.CommonFilePermission)).To(Succeed())

			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "python-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			os.Setenv("ASDF_PYTHON_PATCHES_DIRECTORY", patchDir)
			defer os.Unsetenv("ASDF_PYTHON_PATCHES_DIRECTORY")
			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			err = fixture.plugin.Install(context.Background(), version, downloadDir, installDir)
			Expect(err).NotTo(HaveOccurred())

			pythonBin := filepath.Join(installDir, "bin", "python")
			_, err = os.Stat(pythonBin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("installs without patch when patch file does not exist", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			patchDir, err := os.MkdirTemp("", "python-patches-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(patchDir)

			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "python-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			os.Setenv("ASDF_PYTHON_PATCHES_DIRECTORY", patchDir)
			defer os.Unsetenv("ASDF_PYTHON_PATCHES_DIRECTORY")
			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			err = fixture.plugin.Install(context.Background(), version, downloadDir, installDir)
			Expect(err).NotTo(HaveOccurred())

			pythonBin := filepath.Join(installDir, "bin", "python")
			_, err = os.Stat(pythonBin)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("InstallFromSource [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
		})

		It("downloads and installs from source", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "python-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			err = fixture.plugin.InstallFromSource(context.Background(), version, downloadDir, installDir)
			Expect(err).To(HaveOccurred())

			archivePath := filepath.Join(downloadDir, "Python-"+version+".tgz")
			_, err = os.Stat(archivePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("uses existing archive if present", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			archivePath := filepath.Join(downloadDir, "Python-"+version+".tgz")
			Expect(os.WriteFile(archivePath, []byte("mock archive"), asdf.CommonFilePermission)).To(Succeed())

			installDir, err := os.MkdirTemp("", "python-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			err = fixture.plugin.InstallFromSource(context.Background(), version, downloadDir, installDir)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when external build steps complete", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			downloadDir, err := os.MkdirTemp("", "python-download-success-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "python-install-success-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			archivePath := filepath.Join(downloadDir, "Python-"+version+".tgz")
			Expect(os.WriteFile(archivePath, []byte("mock archive"), asdf.CommonFilePermission)).To(Succeed())

			srcDir := filepath.Join(downloadDir, "Python-"+version)
			Expect(os.MkdirAll(srcDir, asdf.CommonDirectoryPermission)).To(Succeed())

			originalExecFn := execCommandContextFnPython
			defer func() { execCommandContextFnPython = originalExecFn }()

			execCommandContextFnPython = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "true")
			}

			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")
			os.Setenv("ASDF_PYTHON_SKIP_SYSDEPS_CHECK", "1")
			defer os.Unsetenv("ASDF_PYTHON_SKIP_SYSDEPS_CHECK")

			err = fixture.plugin.InstallFromSource(context.Background(), version, downloadDir, installDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Install error handling [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
		})

		It("returns error when patch file cannot be opened", func() {
			version := "3.11.0"
			fixture.SetupVersion(version)

			patchDir, err := os.MkdirTemp("", "python-patches-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(patchDir)

			patchFile := filepath.Join(patchDir, version+".patch")
			Expect(os.WriteFile(patchFile, []byte("mock patch"), 0o000)).To(Succeed())

			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			installDir, err := os.MkdirTemp("", "python-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(installDir)

			os.Setenv("ASDF_PYTHON_PATCHES_DIRECTORY", patchDir)
			defer os.Unsetenv("ASDF_PYTHON_PATCHES_DIRECTORY")
			os.Unsetenv("ASDF_PYTHON_DEFAULT_PACKAGES_FILE")

			err = fixture.plugin.Install(context.Background(), version, downloadDir, installDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("patch"))
		})
	})
})
