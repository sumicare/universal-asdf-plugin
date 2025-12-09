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

package asdf

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Toolchains", func() {
	var (
		origPath string
		tempDir  string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "toolchains-test-*")
		Expect(err).NotTo(HaveOccurred())

		origPath = os.Getenv("PATH")
	})

	AfterEach(func() {
		_ = os.Setenv("PATH", origPath)
		os.RemoveAll(tempDir)
	})

	It("ensures .tool-versions entries for tools", func() {
		homeDir := filepath.Join(tempDir, "home")
		Expect(os.MkdirAll(homeDir, CommonDirectoryPermission)).To(Succeed())
		Expect(os.Setenv("HOME", homeDir)).To(Succeed())

		ctx := context.Background()
		Expect(EnsureToolchains(ctx, "golang")).To(Succeed())

		toolVersionsPath := filepath.Join(homeDir, ".tool-versions")
		data, err := os.ReadFile(toolVersionsPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("golang latest"))
	})

	It("updates a specific .tool-versions file without installing", func() {
		toolVersionsPath := filepath.Join(tempDir, ".tool-versions")

		ctx := context.Background()
		Expect(EnsureToolVersionsFile(ctx, toolVersionsPath, "python")).To(Succeed())

		data, err := os.ReadFile(toolVersionsPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("python latest"))
	})
})
