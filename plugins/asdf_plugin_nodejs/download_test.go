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
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// errForcedNodeArchError simulates a failure in the architecture detector.
var errForcedNodeArchError = errors.New("forced arch error")

var _ = Describe("Download", func() {
	Describe("getNodeArch", func() {
		var originalGetArchFn func() (string, error)

		BeforeEach(func() {
			originalGetArchFn = getArchFnNode
		})

		AfterEach(func() {
			getArchFnNode = originalGetArchFn
		})

		DescribeTable("maps Go arch to Node arch",
			func(goArch, expected string) {
				getArchFnNode = func() (string, error) { return goArch, nil }

				arch, err := getNodeArch()
				Expect(err).NotTo(HaveOccurred())
				Expect(arch).To(Equal(expected))
			},
			Entry("amd64 -> x64", "amd64", "x64"),
			Entry("386 -> x86", "386", "x86"),
			Entry("arm64 -> arm64", "arm64", "arm64"),
			Entry("armv6l -> armv7l", "armv6l", "armv7l"),
			Entry("ppc64 -> ppc64", "ppc64", "ppc64"),
			Entry("ppc64le -> ppc64le", "ppc64le", "ppc64le"),
			Entry("s390x -> s390x", "s390x", "s390x"),
		)

		It("propagates errors from the underlying arch detector", func() {
			getArchFnNode = func() (string, error) {
				return "", errForcedNodeArchError
			}

			arch, err := getNodeArch()
			Expect(err).To(HaveOccurred())
			Expect(arch).To(BeEmpty())
		})
	})

	Describe("verifyNodeChecksum", func() {
		It("verifies correct checksum", func() {
			tmpDir, cleanup := testutil.CreateTestDir(GinkgoT())
			defer cleanup()

			testFile := filepath.Join(tmpDir, "test.tar.gz")
			Expect(os.WriteFile(testFile, []byte("test content"), testutil.CommonFilePermission)).To(Succeed())

			shasumsFile := filepath.Join(tmpDir, "SHASUMS256.txt")
			Expect(os.WriteFile(shasumsFile, []byte("6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72  test.tar.gz\n"),
				testutil.CommonFilePermission)).To(Succeed())

			Expect(verifyNodeChecksum(testFile, shasumsFile, "test.tar.gz")).To(Succeed())
		})

		It("returns error for wrong filename", func() {
			tmpDir, cleanup := testutil.CreateTestDir(GinkgoT())
			defer cleanup()

			testFile := filepath.Join(tmpDir, "test.tar.gz")
			Expect(os.WriteFile(testFile, []byte("test content"), testutil.CommonFilePermission)).To(Succeed())

			shasumsFile := filepath.Join(tmpDir, "SHASUMS256.txt")
			Expect(os.WriteFile(shasumsFile, []byte("6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72  test.tar.gz\n"),
				testutil.CommonFilePermission)).To(Succeed())

			err := verifyNodeChecksum(testFile, shasumsFile, "wrong.tar.gz")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for wrong checksum", func() {
			tmpDir, cleanup := testutil.CreateTestDir(GinkgoT())
			defer cleanup()

			testFile := filepath.Join(tmpDir, "test.tar.gz")
			Expect(os.WriteFile(testFile, []byte("test content"), testutil.CommonFilePermission)).To(Succeed())

			shasumsFile := filepath.Join(tmpDir, "SHASUMS256.txt")
			Expect(os.WriteFile(shasumsFile, []byte("0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"),
				testutil.CommonFilePermission)).To(Succeed())

			err := verifyNodeChecksum(testFile, shasumsFile, "test.tar.gz")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for non-existent shasums file", func() {
			tmpDir, cleanup := testutil.CreateTestDir(GinkgoT())
			defer cleanup()

			testFile := filepath.Join(tmpDir, "test.tar.gz")
			Expect(os.WriteFile(testFile, []byte("test content"), testutil.CommonFilePermission)).To(Succeed())

			err := verifyNodeChecksum(testFile, "/nonexistent/SHASUMS256.txt", "test.tar.gz")
			Expect(err).To(HaveOccurred())
		})
	})

	{
		Describe("Download", func() {
			var fixture *nodeTestFixture

			BeforeEach(func() {
				fixture = newNodeMockFixture()
			})

			AfterEach(func() {
				if fixture != nil {
					fixture.Close()
				}
			})

			It("downloads Node.js", func() {
				version := "20.10.0"
				fixture.SetupVersion(version, "linux", "x64", false)

				downloadDir, err := os.MkdirTemp("", "node-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(downloadDir)

				err = fixture.plugin.Download(context.Background(), version, downloadDir)
				Expect(err).NotTo(HaveOccurred())

				archivePath := filepath.Join(downloadDir, "node.tar.gz")
				_, err = os.Stat(archivePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}

	Describe("Download error cases [mock]", func() {
		var fixture *nodeTestFixture

		BeforeEach(func() {
			fixture = newNodeMockFixture()
		})

		AfterEach(func() {
			if fixture != nil {
				fixture.Close()
			}
		})

		It("downloads Node.js with checksum verification", func() {
			fixture.SetupVersion("20.10.0", "linux", "x64", false)

			downloadDir, err := os.MkdirTemp("", "node-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			Expect(fixture.plugin.Download(context.Background(), "20.10.0", downloadDir)).To(Succeed())

			archivePath := filepath.Join(downloadDir, "node.tar.gz")
			_, err = os.Stat(archivePath)
			Expect(err).NotTo(HaveOccurred())

			shasumsPath := filepath.Join(downloadDir, "SHASUMS256.txt")
			_, err = os.Stat(shasumsPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for non-existent version", func() {
			downloadDir, err := os.MkdirTemp("", "node-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(downloadDir)

			err = fixture.plugin.Download(context.Background(), "99.99.99", downloadDir)
			Expect(err).To(HaveOccurred())
		})
	})
})
