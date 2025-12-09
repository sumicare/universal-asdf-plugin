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
)

var _ = Describe("Download", func() {
	{
		Describe("Download", func() {
			var fixture *goTestFixture

			BeforeEach(func() {
				fixture = newGoMockFixture()
			})

			AfterEach(func() {
				fixture.Close()
			})

			It("downloads Go and verifies archive exists", func() {
				fixture.SetupVersion("1.21.0", "linux", "amd64")

				downloadDir, err := os.MkdirTemp("", "go-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				err = fixture.plugin.Download(context.Background(), "1.21.0", downloadDir)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, "archive.tar.gz")
				_, err = os.Stat(archivePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}

	Describe("Download error cases [mock]", func() {
		var fixture *goTestFixture

		BeforeEach(func() {
			fixture = newGoMockFixture()
		})

		AfterEach(func() {
			fixture.Close()
		})

		It("downloads Go with checksum verification", func() {
			fixture.SetupVersion("1.21.0", "linux", "amd64")

			downloadDir, err := os.MkdirTemp("", "go-download-mock-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			os.Unsetenv("ASDF_GOLANG_SKIP_CHECKSUM")

			err = fixture.plugin.Download(context.Background(), "1.21.0", downloadDir)
			Expect(err).NotTo(HaveOccurred())

			archivePath := filepath.Join(downloadDir, "archive.tar.gz")
			_, err = os.Stat(archivePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for non-existent version", func() {
			downloadDir, err := os.MkdirTemp("", "go-download-mock-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			os.Setenv("ASDF_GOLANG_SKIP_CHECKSUM", "1")
			defer os.Unsetenv("ASDF_GOLANG_SKIP_CHECKSUM")

			err = fixture.plugin.Download(context.Background(), "99.99.99", downloadDir)
			Expect(err).To(HaveOccurred())
		})
	})
})
