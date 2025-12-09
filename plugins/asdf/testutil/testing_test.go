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

package testutil

import (
	"flag"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestData helpers", func() {
	It("creates base test data dir and subdirs", func() {
		cleanupEnv := SetupTestEnv(GinkgoT())
		defer cleanupEnv()

		base := TestDataDir(GinkgoT())
		Expect(base).NotTo(BeEmpty())

		info, err := os.Stat(base)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())

		installDir := TestInstallDir(GinkgoT(), "selftest")
		_, err = os.Stat(installDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Dir(filepath.Dir(installDir))).To(Equal(base))

		downloadDir := TestDownloadDir(GinkgoT(), "selftest")
		_, err = os.Stat(downloadDir)
		Expect(err).NotTo(HaveOccurred())

		buildDir := TestBuildDir(GinkgoT(), "selftest")
		_, err = os.Stat(buildDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("sets and restores ASDF_DATA_DIR", func() {
		original := os.Getenv("ASDF_DATA_DIR")

		cleanup := SetupTestEnv(GinkgoT())
		dataDir := os.Getenv("ASDF_DATA_DIR")
		Expect(dataDir).NotTo(BeEmpty())

		cleanup()

		Expect(os.Getenv("ASDF_DATA_DIR")).To(Equal(original))
	})

	It("creates temp files and directories", func() {
		path, cleanupFile := CreateTestFile(GinkgoT(), "example.txt", []byte("hello"))
		defer cleanupFile()

		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("hello"))

		dir, cleanupDir := CreateTestDir(GinkgoT(), "sub1", "sub2/nested")
		defer cleanupDir()

		_, err = os.Stat(filepath.Join(dir, "sub1"))
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(dir, "sub2", "nested"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns a goldie testdata path with existing parent", func() {
		path := GoldieTestDataPath(GinkgoT())
		Expect(path).NotTo(BeEmpty())

		parent := filepath.Dir(path)
		_, err := os.Stat(parent)
		Expect(err).NotTo(HaveOccurred())
	})

	It("reflects the state of the -update flag in isUpdateMode", func() {
		origFlag := flag.Lookup("update")
		var restore func()

		if origFlag != nil {
			origValue := origFlag.Value.String()
			restore = func() {
				_ = origFlag.Value.Set(origValue) //nolint:errcheck // we're fine with skipping this
			}
		} else {
			f := flag.Bool("update", false, "test flag")
			restore = func() {
				Expect(flag.Set("update", "false")).To(Succeed())
				_ = f
			}
		}

		DeferCleanup(func() {
			if restore != nil {
				restore()
			}
		})

		Expect(flag.Set("update", "false")).To(Succeed())
		Expect(isUpdateMode()).To(BeFalse())

		Expect(flag.Set("update", "true")).To(Succeed())
		Expect(isUpdateMode()).To(BeTrue())
	})

	It("restores an initially empty ASDF_DATA_DIR", func() {
		Expect(os.Unsetenv("ASDF_DATA_DIR")).To(Succeed())

		cleanup := SetupTestEnv(GinkgoT())
		dataDir := os.Getenv("ASDF_DATA_DIR")
		Expect(dataDir).NotTo(BeEmpty())

		cleanup()

		Expect(os.Getenv("ASDF_DATA_DIR")).To(BeEmpty())
	})
})
