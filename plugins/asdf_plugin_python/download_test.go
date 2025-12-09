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
)

var _ = Describe("Download", func() {
	{
		Describe("Download [mock]", func() {
			var fixture *pythonTestFixture

			BeforeEach(func() {
				fixture = newPythonTestFixtureWithMode(true)
			})

			AfterEach(func() {
				if fixture != nil {
					fixture.Close()
				}
			})

			It("downloads Python source", func() {
				version := "3.11.0"
				fixture.SetupVersion(version)

				downloadDir, err := os.MkdirTemp("", "python-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				err = fixture.plugin.DownloadFromFTP(context.Background(), version, downloadDir)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, "Python-"+version+".tgz")
				_, err = os.Stat(archivePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("ensures python-build is available via Download no-op", func() {
				version := "3.11.0"

				downloadDir, err := os.MkdirTemp("", "python-download-noop-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				err = fixture.plugin.Download(context.Background(), version, downloadDir)
				Expect(err).NotTo(HaveOccurred())

				pythonBuildPath := fixture.plugin.pythonBuildPath()
				_, err = os.Stat(pythonBuildPath)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}

	Describe("DownloadFromFTP error cases [mock]", func() {
		var fixture *pythonTestFixture

		BeforeEach(func() {
			fixture = newPythonTestFixtureWithMode(true)
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
		})

		It("returns error for non-existent version", func() {
			downloadDir, err := os.MkdirTemp("", "py-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			err = fixture.plugin.DownloadFromFTP(context.Background(), "0.0.0", downloadDir)
			Expect(err).To(HaveOccurred())
		})

		It("rejects invalid version formats", func() {
			downloadDir, err := os.MkdirTemp("", "python-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			err = fixture.plugin.DownloadFromFTP(context.Background(), "invalid-version", downloadDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("404"))
		})
	})
})
