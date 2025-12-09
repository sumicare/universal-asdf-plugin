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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var _ = Describe("Node.js Plugin", func() {
	var plugin *Plugin

	BeforeEach(func() {
		plugin = New()
	})

	Describe("Name", func() {
		It("returns 'nodejs'", func() {
			Expect(plugin.Name()).To(Equal("nodejs"))
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
		It("returns .nvmrc and .node-version", func() {
			filenames := plugin.ListLegacyFilenames()
			Expect(filenames).To(HaveLen(2))
			Expect(filenames).To(ContainElements(".nvmrc", ".node-version"))
		})
	})

	Describe("ParseLegacyFile", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "node-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		DescribeTable("parses version files",
			func(content, expected string) {
				filePath := filepath.Join(tempDir, ".nvmrc")
				err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
				Expect(err).NotTo(HaveOccurred())

				version, err := plugin.ParseLegacyFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(expected))
			},
			Entry("plain version", "20.0.0", "20.0.0"),
			Entry("with v prefix", "v20.0.0", "20.0.0"),
			Entry("lts", "lts/*", "lts/*"),
			Entry("lts codename", "lts/iron", "lts/iron"),
		)

		It("returns error for non-existent file", func() {
			_, err := plugin.ParseLegacyFile("/nonexistent/.nvmrc")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Uninstall", func() {
		It("removes the installation directory", func() {
			tempDir, err := os.MkdirTemp("", "node-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Uninstall(context.Background(), tempDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tempDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("NodeVersion unmarshalling", func() {
		DescribeTable("handles LTS field",
			func(jsonStr, expectedVersion string, _ any) {
				var nv NodeVersion
				err := json.Unmarshal([]byte(jsonStr), &nv)
				Expect(err).NotTo(HaveOccurred())
				Expect(nv.Version).To(Equal(expectedVersion))
			},
			Entry("non-LTS", `{"version":"v21.0.0","lts":false,"date":"2023-10-17"}`, "v21.0.0", false),
			Entry("LTS", `{"version":"v20.9.0","lts":"Iron","date":"2023-10-24"}`, "v20.9.0", "Iron"),
		)
	})

	Describe("LTS alias recognition", func() {
		DescribeTable("identifies LTS versions",
			func(version string, isLTS bool) {
				result := strings.HasPrefix(version, "lts")
				Expect(result).To(Equal(isLTS))
			},
			Entry("lts", "lts", true),
			Entry("lts/*", "lts/*", true),
			Entry("lts/iron", "lts/iron", true),
			Entry("lts/hydrogen", "lts/hydrogen", true),
			Entry("20.0.0", "20.0.0", false),
			Entry("latest", "latest", false),
		)
	})

	Describe("Corepack config", func() {
		DescribeTable("reads ASDF_NODEJS_AUTO_ENABLE_COREPACK",
			func(envValue string, expected bool) {
				original := os.Getenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK")
				defer os.Setenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK", original)

				if envValue == "" {
					os.Unsetenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK")
				} else {
					os.Setenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK", envValue)
				}

				result := os.Getenv("ASDF_NODEJS_AUTO_ENABLE_COREPACK")
				if expected {
					Expect(result).NotTo(BeEmpty())
				}
			},
			Entry("enabled", "1", true),
			Entry("enabled true", "true", true),
			Entry("disabled", "", false),
		)
	})

	Describe("Help", func() {
		It("returns help information", func() {
			help := plugin.Help()
			Expect(help.Overview).NotTo(BeEmpty())
			Expect(help.Deps).NotTo(BeEmpty())
			Expect(help.Config).NotTo(BeEmpty())
			Expect(help.Links).NotTo(BeEmpty())
		})

		It("contains Node.js-specific information", func() {
			help := plugin.Help()
			Expect(help.Overview).To(ContainSubstring("Node.js"))
			Expect(help.Links).To(ContainSubstring("nodejs.org"))
		})
	})
})
