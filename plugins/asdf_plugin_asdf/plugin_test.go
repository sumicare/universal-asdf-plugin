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

package asdf_plugin_asdf

import (
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// testdataPath returns the path to this plugin's testdata directory.
func testdataPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// pluginTestConfig returns the test configuration for the asdf plugin.
func pluginTestConfig() *testutil.PluginTestConfig {
	return &testutil.PluginTestConfig{
		Config:              &config,
		TestdataPath:        testdataPath(),
		NewPlugin:           New,
		NewPluginWithClient: NewWithClient,
	}
}

var _ = Describe("Asdf Plugin", func() {
	cfg := pluginTestConfig() //nolint:ginkgolinter // we're programmatically bootstrapping test suite
	testutil.DescribeBasicPluginBehavior(cfg)
	testutil.DescribeListAll(cfg)
	testutil.DescribeLatestStable(cfg)
	testutil.DescribeDownload(cfg)
	testutil.DescribeInstall(cfg)
	testutil.DescribeDownloadErrors(cfg)
	testutil.DescribeInstallErrors(cfg)

	Describe("helper functions", func() {
		withEnv := func(key, value string, fn func()) {
			old, had := os.LookupEnv(key)

			if value == "" {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, value)
			}

			DeferCleanup(func() {
				if had {
					_ = os.Setenv(key, old)
				} else {
					_ = os.Unsetenv(key)
				}
			})

			fn()
		}

		It("computes default data dir from HOME when ASDF_DATA_DIR is unset", func() {
			withEnv("ASDF_DATA_DIR", "", func() {
				tmpDir := GinkgoT().TempDir()
				withEnv("HOME", tmpDir, func() {
					got := GetDataDir()
					want := filepath.Join(tmpDir, ".asdf")
					Expect(got).To(Equal(want))
				})
			})
		})

		It("uses ASDF_DATA_DIR when set", func() {
			tmpDir := GinkgoT().TempDir()
			withEnv("ASDF_DATA_DIR", tmpDir, func() {
				Expect(GetDataDir()).To(Equal(tmpDir))
			})
		})

		It("computes shims and plugins directories from ASDF_DATA_DIR", func() {
			base := GinkgoT().TempDir()
			withEnv("ASDF_DATA_DIR", base, func() {
				Expect(GetShimsDir()).To(Equal(filepath.Join(base, "shims")))
				Expect(GetPluginsDir()).To(Equal(filepath.Join(base, "plugins")))
			})
		})

		It("detects whether asdf is installed based on shim existence", func() {
			base := GinkgoT().TempDir()
			withEnv("ASDF_DATA_DIR", base, func() {
				Expect(IsAsdfInstalled()).To(BeFalse())

				shimsDir := filepath.Join(base, "shims")
				Expect(os.MkdirAll(shimsDir, 0o755)).To(Succeed())

				shimPath := filepath.Join(shimsDir, "asdf")
				Expect(os.WriteFile(shimPath, []byte("#!/bin/sh\n"), asdf.CommonFilePermission)).To(Succeed())

				Expect(IsAsdfInstalled()).To(BeTrue())
			})
		})

		It("reports whether asdf shims directory is on PATH", func() {
			base := GinkgoT().TempDir()
			withEnv("ASDF_DATA_DIR", base, func() {
				shimsDir := filepath.Join(base, "shims")

				withEnv("PATH", "/usr/bin"+string(os.PathListSeparator)+"/bin", func() {
					Expect(IsAsdfInPath()).To(BeFalse())
				})

				withEnv("PATH", shimsDir+string(os.PathListSeparator)+"/usr/bin", func() {
					Expect(IsAsdfInPath()).To(BeTrue())
				})
			})
		})

		It("detects shell from SHELL environment variable when set", func() {
			tests := []struct {
				name string
				val  string
				want string
			}{
				{"bash", "/bin/bash", "bash"},
				{"zsh", "/usr/local/bin/zsh", "zsh"},
				{"fish", "/usr/bin/fish", "fish"},
				{"elvish", "/usr/bin/elvish", "elvish"},
				{"nu", "/usr/bin/nu", "nu"},
				{"pwsh", "/usr/bin/pwsh", "pwsh"},
			}

			for i := range tests {
				tc := tests[i]
				By("detecting shell for " + tc.name)
				withEnv("SHELL", tc.val, func() {
					Expect(detectShell()).To(Equal(tc.want))
				})
			}
		})

		It("falls back to OS-specific default shell when SHELL is unset", func() {
			withEnv("SHELL", "", func() {
				got := detectShell()
				if runtime.GOOS == "windows" {
					Expect(got).To(Equal("pwsh"))
				} else {
					Expect(got).To(Equal("bash"))
				}
			})
		})

		It("returns shell-specific configuration instructions", func() {
			cases := []struct {
				name string
				arg  string
				want string
			}{
				{"bash", "bash", "~/.bashrc"},
				{"zsh", "zsh", "~/.zshrc"},
				{"fish", "fish", "config.fish"},
				{"pwsh", "pwsh", "profile.ps1"},
				{"powershell", "powershell", "profile.ps1"},
				{"nu", "nu", "config.nu"},
				{"nushell", "nushell", "config.nu"},
				{"elvish", "elvish", "rc.elv"},
				{"default", "unknown-shell", "asdf shims to your PATH"},
			}

			for i := range cases {
				c := cases[i]
				By("validating instructions for " + c.name)
				got := GetShellConfigInstructions(c.arg)
				Expect(got).To(ContainSubstring(c.want))
			}
		})

		It("prints shell configuration help for common shells", func() {
			cases := []struct {
				name  string
				shell string
			}{
				{"bash", "/bin/bash"},
				{"zsh", "/usr/local/bin/zsh"},
				{"fish", "/usr/bin/fish"},
				{"default", "/usr/bin/unknown-shell"},
			}

			for i := range cases {
				c := cases[i]
				By("printing shell config help for " + c.name)
				withEnv("SHELL", c.shell, func() {
					printShellConfigHelp()
				})
			}
		})
	})
})
