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
			tempDir, err := os.MkdirTemp("", "go-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Unsetenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), "1.21.0", tempDir)).To(Succeed())
		})

		It("returns nil when custom file does not exist", func() {
			tempDir, err := os.MkdirTemp("", "go-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Setenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE", "/nonexistent/file")
			defer os.Unsetenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE")
			Expect(plugin.installDefaultPackages(context.Background(), "1.21.0", tempDir)).To(Succeed())
		})

		It("reads packages from file with comments and empty lines", func() {
			tempDir, err := os.MkdirTemp("", "go-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			pkgFile := filepath.Join(tempDir, "pkgs")
			err = os.WriteFile(pkgFile, []byte("// comment\n\ngolang.org/x/tools/gopls@latest // inline comment\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE", pkgFile)
			defer os.Unsetenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE")

			Expect(plugin.installDefaultPackages(context.Background(), "1.21.0", tempDir)).To(Succeed())
		})

		It("handles old Go versions with go get", func() {
			tempDir, err := os.MkdirTemp("", "go-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			pkgFile := filepath.Join(tempDir, "pkgs")
			err = os.WriteFile(pkgFile, []byte("github.com/example/pkg\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE", pkgFile)
			defer os.Unsetenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE")

			Expect(plugin.installDefaultPackages(context.Background(), "1.15.0", tempDir)).To(Succeed())
		})
	})

	{
		Describe("Install", func() {
			var fixture *goTestFixture

			BeforeEach(func() {
				fixture = newGoMockFixture()
			})

			AfterEach(func() {
				fixture.Close()
			})

			It("downloads and installs Go", func() {
				fixture.SetupVersion("1.21.0", "linux", "amd64")

				downloadDir, err := os.MkdirTemp("", "go-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "go-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				os.Unsetenv("ASDF_GOLANG_DEFAULT_PACKAGES_FILE")

				err = fixture.plugin.Download(context.Background(), "1.21.0", downloadDir)
				Expect(err).NotTo(HaveOccurred())

				err = fixture.plugin.Install(context.Background(), "1.21.0", downloadDir, installDir)
				Expect(err).NotTo(HaveOccurred())

				goBin := filepath.Join(installDir, "go", "bin", "go")
				_, err = os.Stat(goBin)
				Expect(err).NotTo(HaveOccurred())

				cmd := exec.Command(goBin, "version")
				cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
				out, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), "failed to run go version: %s", string(out))
				Expect(string(out)).To(ContainSubstring("go1.21.0"))
			})

			It("auto-downloads if archive missing", func() {
				fixture.SetupVersion("1.21.0", "linux", "amd64")

				downloadDir, err := os.MkdirTemp("", "go-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				installDir, err := os.MkdirTemp("", "go-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(installDir)

				err = fixture.plugin.Install(context.Background(), "1.21.0", downloadDir, installDir)
				Expect(err).NotTo(HaveOccurred())

				goBin := filepath.Join(installDir, "go", "bin", "go")
				_, err = os.Stat(goBin)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}
})
