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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var _ = Describe("Go Plugin", func() {
	var plugin *Plugin

	BeforeEach(func() {
		plugin = New()
	})

	Describe("Name", func() {
		It("returns 'golang'", func() {
			Expect(plugin.Name()).To(Equal("golang"))
		})
	})

	Describe("ListBinPaths", func() {
		It("returns correct bin paths", func() {
			Expect(plugin.ListBinPaths()).To(Equal("go/bin bin"))
		})
	})

	Describe("ExecEnv", func() {
		It("returns GOROOT, GOPATH, and GOBIN when not set", func() {
			os.Unsetenv("GOROOT")
			os.Unsetenv("GOPATH")
			os.Unsetenv("GOBIN")

			env := plugin.ExecEnv("/tmp/install")
			Expect(env["GOROOT"]).To(Equal("/tmp/install/go"))
			Expect(env["GOPATH"]).To(Equal("/tmp/install/packages"))
			Expect(env["GOBIN"]).To(Equal("/tmp/install/bin"))
		})

		It("respects user-set GOROOT", func() {
			os.Setenv("GOROOT", "/custom/goroot")
			defer os.Unsetenv("GOROOT")
			os.Unsetenv("GOPATH")
			os.Unsetenv("GOBIN")

			env := plugin.ExecEnv("/tmp/install")
			Expect(env).NotTo(HaveKey("GOROOT"))
			Expect(env["GOPATH"]).To(Equal("/tmp/install/packages"))
		})

		It("respects user-set GOPATH", func() {
			os.Unsetenv("GOROOT")
			os.Setenv("GOPATH", "/custom/gopath")
			defer os.Unsetenv("GOPATH")
			os.Unsetenv("GOBIN")

			env := plugin.ExecEnv("/tmp/install")
			Expect(env["GOROOT"]).To(Equal("/tmp/install/go"))
			Expect(env).NotTo(HaveKey("GOPATH"))
		})

		It("respects user-set GOBIN", func() {
			os.Unsetenv("GOROOT")
			os.Unsetenv("GOPATH")
			os.Setenv("GOBIN", "/custom/gobin")
			defer os.Unsetenv("GOBIN")

			env := plugin.ExecEnv("/tmp/install")
			Expect(env).NotTo(HaveKey("GOBIN"))
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns .go-version, go.mod, and go.work", func() {
			filenames := plugin.ListLegacyFilenames()
			Expect(filenames).To(HaveLen(3))
			Expect(filenames).To(ContainElements(".go-version", "go.mod", "go.work"))
		})
	})

	Describe("ParseLegacyFile", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "go-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		DescribeTable("parses .go-version files",
			func(content, expected string) {
				filePath := filepath.Join(tempDir, ".go-version")
				err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
				Expect(err).NotTo(HaveOccurred())

				version, err := plugin.ParseLegacyFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(expected))
			},
			Entry("plain version", "1.21.0", "1.21.0"),
			Entry("with go prefix", "go1.21.0", "1.21.0"),
			Entry("with newline", "1.21.0\n", "1.21.0"),
			Entry("with spaces", "  1.21.0  ", "1.21.0"),
		)

		DescribeTable("parses go.mod files",
			func(content, expected string) {
				filePath := filepath.Join(tempDir, "go.mod")
				err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
				Expect(err).NotTo(HaveOccurred())

				version, err := plugin.ParseLegacyFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(expected))
			},
			Entry("simple go.mod", "module example.com/test\n\ngo 1.21\n", "1.21"),
			Entry("go.mod with patch version", "module example.com/test\n\ngo 1.21.5\n", "1.21.5"),
			Entry("go.mod with require", "module example.com/test\n\ngo 1.22\n\nrequire (\n\tgithub.com/pkg v1.0.0\n)\n", "1.22"),
			Entry("heroku format", "module example.com/test\n// +heroku goVersion go1.20\n", "1.20"),
		)

		DescribeTable("parses go.work files",
			func(content, expected string) {
				filePath := filepath.Join(tempDir, "go.work")
				err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
				Expect(err).NotTo(HaveOccurred())

				version, err := plugin.ParseLegacyFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(expected))
			},
			Entry("simple go.work", "go 1.21\n\nuse (\n\t./app\n)\n", "1.21"),
			Entry("go.work with patch", "go 1.22.1\n\nuse ./app\n", "1.22.1"),
		)

		It("returns error for nonexistent file", func() {
			_, err := plugin.ParseLegacyFile("/nonexistent/file")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for go.mod without version", func() {
			filePath := filepath.Join(tempDir, "go.mod")
			err := os.WriteFile(filePath, []byte("module example.com/test\n"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			_, err = plugin.ParseLegacyFile(filePath)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Uninstall", func() {
		It("removes the installation directory", func() {
			tempDir, err := os.MkdirTemp("", "go-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())

			testFile := filepath.Join(tempDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Uninstall(context.Background(), tempDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tempDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("Help", func() {
		It("returns help information", func() {
			help := plugin.Help()
			Expect(help.Overview).NotTo(BeEmpty())
			Expect(help.Deps).NotTo(BeEmpty())
			Expect(help.Config).NotTo(BeEmpty())
			Expect(help.Links).NotTo(BeEmpty())
		})

		It("contains Go-specific information", func() {
			help := plugin.Help()
			Expect(help.Overview).To(ContainSubstring("Go"))
			Expect(help.Links).To(ContainSubstring("go.dev"))
		})
	})
})
