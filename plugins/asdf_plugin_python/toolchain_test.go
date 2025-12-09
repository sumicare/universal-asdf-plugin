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
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	// errHomeDirLookupFailed simulates a home directory lookup failure in Python toolchain tests.
	errHomeDirLookupFailed = errors.New("home dir lookup failed")
	// errDownloadMkdirFailed simulates a download directory creation failure in Python toolchain tests.
	errDownloadMkdirFailed = errors.New("download mkdir failed")
	// errInstallMkdirFailed simulates an install directory creation failure in Python toolchain tests.
	errInstallMkdirFailed = errors.New("install mkdir failed")
)

var _ = Describe("Python toolchain helpers", func() {
	Describe("InstallPythonToolchain", func() {
		var (
			origUserHomeFn func() (string, error)
			origMkdirFn    func(string, os.FileMode) error
			origNewPlugin  func() *Plugin
			origLatestFn   func(*Plugin, context.Context, string) (string, error)
			origInstallFn  func(*Plugin, context.Context, string, string, string) error
			tempDir        string
		)

		BeforeEach(func() {
			origUserHomeFn = userHomeDirFnPython
			origMkdirFn = mkdirAllFnPython
			origNewPlugin = newPythonPluginFnPython
			origLatestFn = pythonLatestStableFn
			origInstallFn = pythonInstallFn

			var err error
			tempDir, err = os.MkdirTemp("", "python-toolchain-test-*")
			Expect(err).NotTo(HaveOccurred())
			userHomeDirFnPython = func() (string, error) { return tempDir, nil }
		})

		AfterEach(func() {
			userHomeDirFnPython = origUserHomeFn
			mkdirAllFnPython = origMkdirFn
			newPythonPluginFnPython = origNewPlugin
			pythonLatestStableFn = origLatestFn
			pythonInstallFn = origInstallFn
			_ = os.RemoveAll(tempDir)
		})

		It("returns error when home directory cannot be determined", func() {
			os.Unsetenv("ASDF_DATA_DIR")
			userHomeDirFnPython = func() (string, error) {
				return "", errHomeDirLookupFailed
			}

			err := InstallPythonToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("determining home directory for ASDF_DATA_DIR fallback"))
		})

		It("returns error when creating Python download directory fails", func() {
			os.Unsetenv("ASDF_DATA_DIR")

			mkdirAllFnPython = func(path string, perm os.FileMode) error {
				parent := filepath.Dir(path)
				grandparent := filepath.Dir(parent)
				if filepath.Base(grandparent) == "downloads" {
					return errDownloadMkdirFailed
				}

				return origMkdirFn(path, perm)
			}

			err := InstallPythonToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating download directory for python"))
		})

		It("returns error when creating Python install directory fails", func() {
			os.Unsetenv("ASDF_DATA_DIR")

			mkdirAllFnPython = func(path string, perm os.FileMode) error {
				parent := filepath.Dir(path)
				grandparent := filepath.Dir(parent)
				if filepath.Base(grandparent) == "installs" {
					return errInstallMkdirFailed
				}

				return origMkdirFn(path, perm)
			}

			err := InstallPythonToolchain(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating install directory for python"))
		})

		It("installs Python toolchain via seams successfully", func() {
			os.Unsetenv("ASDF_DATA_DIR")

			createdPlugin := &Plugin{}
			newPythonPluginFnPython = func() *Plugin {
				return createdPlugin
			}

			pythonLatestStableFn = func(p *Plugin, _ context.Context, query string) (string, error) {
				Expect(p).To(Equal(createdPlugin))
				Expect(query).To(Equal(""))

				return "3.11.0", nil
			}

			calledInstall := false
			pythonInstallFn = func(p *Plugin, _ context.Context, version, downloadPath, installPath string) error {
				calledInstall = true
				Expect(p).To(Equal(createdPlugin))
				Expect(version).To(Equal("3.11.0"))
				Expect(downloadPath).To(ContainSubstring("downloads/python/3.11.0"))
				Expect(installPath).To(ContainSubstring("installs/python/3.11.0"))

				return nil
			}

			Expect(InstallPythonToolchain(context.Background())).To(Succeed())
			Expect(calledInstall).To(BeTrue())
		})
	})
})
