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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var _ = Describe("Python Plugin", func() {
	var plugin *Plugin

	BeforeEach(func() {
		plugin = New()
	})

	Describe("Constructors", func() {
		It("creates plugin with New", func() {
			p := New()
			Expect(p).NotTo(BeNil())
			Expect(p.ftpURL).To(Equal(pythonFTPURL))
			Expect(p.githubClient).NotTo(BeNil())
		})

		It("creates plugin with NewWithURLs", func() {
			customFTP := "https://custom.ftp.url/"
			p := NewWithURLs(customFTP, nil)
			Expect(p).NotTo(BeNil())
			Expect(p.ftpURL).To(Equal(customFTP))
		})

		It("creates plugin with NewWithBuildDir", func() {
			customDir := "/custom/build/dir"
			p := NewWithBuildDir(customDir)
			Expect(p).NotTo(BeNil())
			Expect(p.pyenvDir).To(Equal(customDir))
			Expect(p.ftpURL).To(Equal(pythonFTPURL))
			Expect(p.githubClient).NotTo(BeNil())
		})
	})

	Describe("Name", func() {
		It("returns 'python'", func() {
			Expect(plugin.Name()).To(Equal("python"))
		})
	})

	Describe("ListBinPaths", func() {
		It("returns 'bin'", func() {
			Expect(plugin.ListBinPaths()).To(Equal("bin"))
		})
	})

	Describe("ExecEnv", func() {
		It("returns nil (no special env vars)", func() {
			env := plugin.ExecEnv("/tmp/install")
			Expect(env).To(BeNil())
		})
	})

	Describe("ListLegacyFilenames", func() {
		It("returns .python-version", func() {
			filenames := plugin.ListLegacyFilenames()
			Expect(filenames).To(HaveLen(1))
			Expect(filenames[0]).To(Equal(".python-version"))
		})
	})

	Describe("ParseLegacyFile", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "python-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		DescribeTable("parses version files",
			func(content, expected string) {
				filePath := filepath.Join(tempDir, ".python-version")
				err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
				Expect(err).NotTo(HaveOccurred())

				version, err := plugin.ParseLegacyFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(expected))
			},
			Entry("plain version", "3.11.0", "3.11.0"),
			Entry("with newline", "3.11.0\n", "3.11.0"),
			Entry("with spaces", "  3.11.0  ", "3.11.0"),
		)
	})

	Describe("Uninstall", func() {
		It("removes the installation directory", func() {
			tempDir, err := os.MkdirTemp("", "python-plugin-test-*")
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

		It("contains Python-specific information", func() {
			help := plugin.Help()
			Expect(help.Overview).To(ContainSubstring("Python"))
			Expect(help.Links).To(ContainSubstring("python.org"))
		})
	})
})
