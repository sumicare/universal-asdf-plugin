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
)

var (
	// errMkdirFailed simulates a mkdir failure in toolchain tests.
	errMkdirFailed = errors.New("mkdir failed")
	// errHomeDirLookupFailed simulates a home directory lookup failure in toolchain tests.
	errHomeDirLookupFailed = errors.New("home dir lookup failed")
)

var _ = Describe("Node.js toolchain helpers", func() {
	Describe("EnsureNodeToolchainEntries", func() {
		It("delegates to EnsureToolchains for nodejs", func() {
			called := false
			original := ensureToolchainsFnNode
			defer func() { ensureToolchainsFnNode = original }()

			ensureToolchainsFnNode = func(_ context.Context, tools ...string) error {
				called = true
				Expect(tools).To(ContainElement("nodejs"))

				return nil
			}

			Expect(EnsureNodeToolchainEntries(context.Background())).To(Succeed())
			Expect(called).To(BeTrue())
		})
	})

	Describe("InstallNodeToolchain", func() {
		var (
			origUserHome func() (string, error)
			tempDir      string
		)

		BeforeEach(func() {
			origUserHome = userHomeDirFnNode
			var err error
			tempDir, err = os.MkdirTemp("", "node-toolchain-test-*")
			Expect(err).NotTo(HaveOccurred())
			userHomeDirFnNode = func() (string, error) { return tempDir, nil }
		})

		AfterEach(func() {
			userHomeDirFnNode = origUserHome
			_ = os.RemoveAll(tempDir)
		})

		simulateMkdirFailure := func(original func(string, os.FileMode) error, grandparentBase string) func(string, os.FileMode) error {
			return func(path string, perm os.FileMode) error {
				parent := filepath.Dir(path)
				grandparent := filepath.Dir(parent)
				if filepath.Base(grandparent) == grandparentBase {
					return errMkdirFailed
				}

				return original(path, perm)
			}
		}

		It("returns error when home directory cannot be determined", func() {
			userHomeDirFnNode = func() (string, error) {
				return "", errHomeDirLookupFailed
			}

			err := InstallNodeToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("determining home directory for ASDF_DATA_DIR fallback"))
		})

		It("returns error when creating Node download directory fails", func() {
			originalMkdir := mkdirAllFnNode
			defer func() { mkdirAllFnNode = originalMkdir }()

			mkdirAllFnNode = simulateMkdirFailure(originalMkdir, "downloads")

			err := InstallNodeToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating download directory for nodejs"))
		})

		It("returns error when creating Node install directory fails", func() {
			originalMkdir := mkdirAllFnNode
			defer func() { mkdirAllFnNode = originalMkdir }()

			mkdirAllFnNode = simulateMkdirFailure(originalMkdir, "installs")

			err := InstallNodeToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating install directory for nodejs"))
		})
	})
})
