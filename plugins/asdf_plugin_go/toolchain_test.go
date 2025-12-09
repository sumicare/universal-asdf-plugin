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

var _ = Describe("Go toolchain helpers", func() {
	Describe("EnsureGoToolchainEntries", func() {
		It("delegates to EnsureToolchains for golang", func() {
			called := false
			original := ensureToolchainsFnGo
			defer func() { ensureToolchainsFnGo = original }()

			ensureToolchainsFnGo = func(_ context.Context, tools ...string) error {
				called = true
				Expect(tools).To(ContainElement("golang"))

				return nil
			}

			Expect(EnsureGoToolchainEntries(context.Background())).To(Succeed())
			Expect(called).To(BeTrue())
		})
	})

	Describe("InstallGoToolchain", func() {
		var (
			origUserHome func() (string, error)
			tempDir      string
		)

		BeforeEach(func() {
			origUserHome = userHomeDirFnGo
			var err error
			tempDir, err = os.MkdirTemp("", "go-toolchain-test-*")
			Expect(err).NotTo(HaveOccurred())
			userHomeDirFnGo = func() (string, error) { return tempDir, nil }
		})

		AfterEach(func() {
			userHomeDirFnGo = origUserHome
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
			userHomeDirFnGo = func() (string, error) {
				return "", errHomeDirLookupFailed
			}

			err := InstallGoToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("determining home directory for ASDF_DATA_DIR fallback"))
		})

		It("returns error when creating Go download directory fails", func() {
			originalMkdir := mkdirAllFnGo
			defer func() { mkdirAllFnGo = originalMkdir }()

			mkdirAllFnGo = simulateMkdirFailure(originalMkdir, "downloads")

			err := InstallGoToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating download directory for golang"))
		})

		It("returns error when creating Go install directory fails", func() {
			originalMkdir := mkdirAllFnGo
			defer func() { mkdirAllFnGo = originalMkdir }()

			mkdirAllFnGo = simulateMkdirFailure(originalMkdir, "installs")

			err := InstallGoToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating install directory for golang"))
		})
	})
})
