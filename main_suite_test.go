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

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/plugins"
)

// TestMainSuite runs the top-level CLI test suite.
func TestMainSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

// testEnvVarValidation is a shared helper for testing environment variable
// validation in cmdDownload and cmdInstall. It validates that missing
// ASDF_INSTALL_VERSION and the given pathEnvVar both produce errors.
func testEnvVarValidation(pathEnvVar, pathValue string, cmdFunc func(asdf.Plugin) error) {
	var originalVersion, originalPath string

	BeforeEach(func() {
		originalVersion = os.Getenv("ASDF_INSTALL_VERSION")
		originalPath = os.Getenv(pathEnvVar)
	})

	AfterEach(func() {
		os.Setenv("ASDF_INSTALL_VERSION", originalVersion)
		os.Setenv(pathEnvVar, originalPath)
	})

	It("returns error when ASDF_INSTALL_VERSION is missing", func() {
		os.Unsetenv("ASDF_INSTALL_VERSION")
		os.Setenv(pathEnvVar, pathValue)

		plugin, err := plugins.GetPlugin("golang")
		Expect(err).NotTo(HaveOccurred())

		err = cmdFunc(plugin)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ASDF_INSTALL_VERSION"))
	})

	It("returns error when "+pathEnvVar+" is missing", func() {
		os.Setenv("ASDF_INSTALL_VERSION", "1.21.0")
		os.Unsetenv(pathEnvVar)

		plugin, err := plugins.GetPlugin("golang")
		Expect(err).NotTo(HaveOccurred())

		err = cmdFunc(plugin)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(pathEnvVar))
	})
}

// Main CLI Ginkgo entry point exercises the CLI wiring end-to-end using
// real plugin implementations and a small mock plugin.
var _ = Describe("Main CLI", func() {
	Describe("getPlugin", func() {
		DescribeTable("returns correct plugin",
			func(name, expectedName string, expectError bool) {
				plugin, err := plugins.GetPlugin(name)
				if expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(plugin.Name()).To(Equal(expectedName))
				}
			},
			Entry("golang", "golang", "golang", false),
			Entry("go alias", "go", "golang", false),
			Entry("python", "python", "python", false),
			Entry("nodejs", "nodejs", "nodejs", false),
			Entry("node alias", "node", "nodejs", false),
			Entry("argo", "argo", "argo", false),
			Entry("argocd", "argocd", "argocd", false),
			Entry("argo-rollouts", "argo-rollouts", "argo-rollouts", false),
			Entry("checkov", "checkov", "checkov", false),
			Entry("cmake", "cmake", "cmake", false),
			Entry("cosign", "cosign", "cosign", false),
			Entry("doctl", "doctl", "doctl", false),
			Entry("jq", "jq", "jq", false),
			Entry("k9s", "k9s", "k9s", false),
			Entry("kind", "kind", "kind", false),
			Entry("ko", "ko", "ko", false),
			Entry("kubectl", "kubectl", "kubectl", false),
			Entry("lazygit", "lazygit", "lazygit", false),
			Entry("linkerd", "linkerd", "linkerd", false),
			Entry("nerdctl", "nerdctl", "nerdctl", false),
			Entry("ginkgo", "ginkgo", "ginkgo", false),
			Entry("github-cli", "github-cli", "github-cli", false),
			Entry("gh alias", "gh", "github-cli", false),
			Entry("gitsign", "gitsign", "gitsign", false),
			Entry("gitleaks", "gitleaks", "gitleaks", false),
			Entry("goreleaser", "goreleaser", "goreleaser", false),
			Entry("golangci-lint", "golangci-lint", "golangci-lint", false),
			Entry("grype", "grype", "grype", false),
			Entry("sccache", "sccache", "sccache", false),
			Entry("shellcheck", "shellcheck", "shellcheck", false),
			Entry("sops", "sops", "sops", false),
			Entry("shfmt", "shfmt", "shfmt", false),
			Entry("syft", "syft", "syft", false),
			Entry("terraform", "terraform", "terraform", false),
			Entry("terragrunt", "terragrunt", "terragrunt", false),
			Entry("terrascan", "terrascan", "terrascan", false),
			Entry("tfupdate", "tfupdate", "tfupdate", false),
			Entry("tflint", "tflint", "tflint", false),
			Entry("trivy", "trivy", "trivy", false),
			Entry("vultr-cli", "vultr-cli", "vultr-cli", false),
			Entry("opentofu", "opentofu", "opentofu", false),
			Entry("protoc", "protoc", "protoc", false),
			Entry("upx", "upx", "upx", false),
			Entry("uv", "uv", "uv", false),
			Entry("yq", "yq", "yq", false),
			Entry("zig", "zig", "zig", false),
			Entry("unknown", "unknown", "", true),
		)
	})

	Describe("printUsage", func() {
		It("returns no error", func() {
			err := printUsage()
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when latest-stable fails", func() {
			os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
			mock := &mockPlugin{name: "mock", latestStableErr: true}

			err := cmdLatestStable(context.Background(), mock)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("run", func() {
		var originalArgs []string

		BeforeEach(func() {
			originalArgs = os.Args
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally restore os.Args
		})

		assertRunSucceeds := func(args ...string) {
			os.Args = args //nolint:reassign // tests intentionally override os.Args
			Expect(run()).To(Succeed())
		}

		Describe("global commands", func() {
			It("handles no args", func() { assertRunSucceeds("cmd") })
			It("handles version", func() { assertRunSucceeds("cmd", "version") })
			It("handles help", func() { assertRunSucceeds("cmd", "help") })
			It("handles plugins", func() { assertRunSucceeds("cmd", "plugins") })
			It("handles --version", func() { assertRunSucceeds("cmd", "--version") })
			It("handles -h", func() { assertRunSucceeds("cmd", "-h") })
		})

		Describe("plugin commands", func() {
			withPluginEnv := func() {
				os.Setenv("ASDF_PLUGIN_NAME", "golang")
			}

			AfterEach(func() {
				os.Unsetenv("ASDF_PLUGIN_NAME")
			})

			It("handles list-bin-paths", func() {
				withPluginEnv()
				assertRunSucceeds("cmd", "list-bin-paths")
			})

			It("handles list-legacy-filenames", func() {
				withPluginEnv()
				assertRunSucceeds("cmd", "list-legacy-filenames")
			})

			It("handles help subcommands", func() {
				withPluginEnv()
				for _, cmd := range []string{"help.overview", "help.deps", "help.config", "help.links"} {
					assertRunSucceeds("cmd", cmd)
				}
			})
		})

		It("handles exec-env", func() {
			os.Args = []string{"cmd", "exec-env"} //nolint:reassign // tests intentionally override os.Args
			os.Setenv("ASDF_PLUGIN_NAME", "golang")
			os.Setenv("ASDF_INSTALL_PATH", "/tmp/test")
			defer os.Unsetenv("ASDF_PLUGIN_NAME")
			defer os.Unsetenv("ASDF_INSTALL_PATH")
			Expect(run()).To(Succeed())
		})

		It("handles uninstall", func() {
			tempDir, err := os.MkdirTemp("", "test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			os.Args = []string{"cmd", "uninstall"} //nolint:reassign // tests intentionally override os.Args
			os.Setenv("ASDF_PLUGIN_NAME", "golang")
			os.Setenv("ASDF_INSTALL_PATH", tempDir)
			defer os.Unsetenv("ASDF_PLUGIN_NAME")
			defer os.Unsetenv("ASDF_INSTALL_PATH")
			Expect(run()).To(Succeed())
		})

		It("handles parse-legacy-file", func() {
			tempDir, err := os.MkdirTemp("", "test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)
			f := filepath.Join(tempDir, ".go-version")
			err = os.WriteFile(f, []byte("1.21.0"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())
			os.Args = []string{"cmd", "parse-legacy-file", f} //nolint:reassign // tests intentionally override os.Args
			os.Setenv("ASDF_PLUGIN_NAME", "golang")
			defer os.Unsetenv("ASDF_PLUGIN_NAME")
			Expect(run()).To(Succeed())
		})

		Describe("plugin detection from executable name", func() {
			It("detects plugins by executable prefix", func() {
				for _, exe := range []string{"asdf-golang", "asdf-python", "asdf-nodejs"} {
					os.Args = []string{exe, "list-bin-paths"} //nolint:reassign // tests intentionally override os.Args
					os.Unsetenv("ASDF_PLUGIN_NAME")
					Expect(run()).To(Succeed())
				}
			})
		})

		It("returns error for unknown plugin", func() {
			os.Args = []string{"cmd", "list-all"} //nolint:reassign // tests intentionally override os.Args
			os.Setenv("ASDF_PLUGIN_NAME", "unknown")
			defer os.Unsetenv("ASDF_PLUGIN_NAME")
			Expect(run()).ToNot(Succeed())
		})

		It("returns error for unknown command", func() {
			os.Args = []string{"cmd", "unknown"} //nolint:reassign // tests intentionally override os.Args
			os.Setenv("ASDF_PLUGIN_NAME", "golang")
			defer os.Unsetenv("ASDF_PLUGIN_NAME")
			Expect(run()).ToNot(Succeed())
		})

		It("returns error when no plugin specified", func() {
			os.Args = []string{"cmd", "list-bin-paths"} //nolint:reassign // tests intentionally override os.Args
			os.Unsetenv("ASDF_PLUGIN_NAME")
			err := run()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin name required"))
		})
	})

	Describe("cmdListBinPaths", func() {
		DescribeTable("returns bin paths for each plugin",
			func(pluginName string) {
				plugin, err := plugins.GetPlugin(pluginName)
				Expect(err).NotTo(HaveOccurred())

				err = cmdListBinPaths(plugin)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("golang", "golang"),
			Entry("python", "python"),
			Entry("nodejs", "nodejs"),
		)
	})

	Describe("cmdExecEnv", func() {
		var originalPath string

		BeforeEach(func() {
			originalPath = os.Getenv("ASDF_INSTALL_PATH")
		})

		AfterEach(func() {
			os.Setenv("ASDF_INSTALL_PATH", originalPath)
		})

		It("returns no error with install path set", func() {
			os.Setenv("ASDF_INSTALL_PATH", "/tmp/install")
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdExecEnv(plugin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns no error with empty install path", func() {
			os.Unsetenv("ASDF_INSTALL_PATH")
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdExecEnv(plugin)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("cmdDownload", func() {
		testEnvVarValidation("ASDF_DOWNLOAD_PATH", "/tmp/download",
			func(plugin asdf.Plugin) error {
				return cmdDownload(context.Background(), plugin)
			})
	})

	Describe("cmdInstall", func() {
		testEnvVarValidation("ASDF_INSTALL_PATH", "/tmp/install",
			func(plugin asdf.Plugin) error {
				return cmdInstall(context.Background(), plugin)
			})

		It("returns error when download path is missing and version empty", func() {
			os.Unsetenv("ASDF_INSTALL_VERSION")
			os.Unsetenv("ASDF_INSTALL_PATH")
			os.Unsetenv("ASDF_DOWNLOAD_PATH")

			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			installErr := cmdInstall(context.Background(), plugin)
			Expect(installErr).To(HaveOccurred())
		})
	})

	Describe("cmdUninstall", func() {
		var originalPath string

		BeforeEach(func() {
			originalPath = os.Getenv("ASDF_INSTALL_PATH")
		})

		AfterEach(func() {
			os.Setenv("ASDF_INSTALL_PATH", originalPath)
		})

		It("returns error when ASDF_INSTALL_PATH is missing", func() {
			os.Unsetenv("ASDF_INSTALL_PATH")
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdUninstall(context.Background(), plugin)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ASDF_INSTALL_PATH"))
		})

		It("removes directory when path is set", func() {
			tempDir, err := os.MkdirTemp("", "uninstall-test-*")
			Expect(err).NotTo(HaveOccurred())

			os.Setenv("ASDF_INSTALL_PATH", tempDir)
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdUninstall(context.Background(), plugin)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tempDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("cmdListLegacyFilenames", func() {
		DescribeTable("returns legacy filenames for each plugin",
			func(pluginName string) {
				plugin, err := plugins.GetPlugin(pluginName)
				Expect(err).NotTo(HaveOccurred())

				err = cmdListLegacyFilenames(plugin)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("golang", "golang"),
			Entry("python", "python"),
			Entry("nodejs", "nodejs"),
		)
	})

	Describe("cmdParseLegacyFile", func() {
		var tempDir string
		var originalArgs []string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "legacy-file-test-*")
			Expect(err).NotTo(HaveOccurred())
			originalArgs = os.Args
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
		})

		It("parses legacy file successfully", func() {
			legacyFile := filepath.Join(tempDir, ".go-version")
			err := os.WriteFile(legacyFile, []byte("1.21.0"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			os.Args = []string{"cmd", "parse-legacy-file", legacyFile} //nolint:reassign // tests intentionally override os.Args
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdParseLegacyFile(plugin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when file argument is missing", func() {
			os.Args = []string{"cmd", "parse-legacy-file"} //nolint:reassign // tests intentionally override os.Args
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdParseLegacyFile(plugin)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when file does not exist", func() {
			os.Args = []string{"cmd", "parse-legacy-file", "/nonexistent/file"} //nolint:reassign // tests intentionally override os.Args
			plugin, err := plugins.GetPlugin("golang")
			Expect(err).NotTo(HaveOccurred())

			err = cmdParseLegacyFile(plugin)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("cmdUpdateToolVersions", func() {
		var (
			originalArgs []string
			originalCwd  string
			tempDir      string
		)

		BeforeEach(func() {
			var err error
			originalArgs = os.Args
			originalCwd, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())

			tempDir, err = os.MkdirTemp("", "update-tool-versions-*")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tempDir)).To(Succeed())
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
			Expect(os.Chdir(originalCwd)).To(Succeed())
			os.RemoveAll(tempDir)
		})

		It("returns nil when no tools are defined", func() {
			os.Args = []string{"cmd", "update-tool-versions"} //nolint:reassign // tests intentionally override os.Args

			Expect(os.WriteFile(".tool-versions", []byte("# empty\n"), asdf.CommonFilePermission)).To(Succeed())

			Expect(cmdUpdateToolVersions()).To(Succeed())
		})

		It("propagates parse error for missing file", func() {
			os.Args = []string{"cmd", "update-tool-versions", "nonexistent"} //nolint:reassign // tests intentionally override os.Args

			err := cmdUpdateToolVersions()
			Expect(err).To(HaveOccurred())
		})

		When("running online tests", func() {
			BeforeEach(func() {
				if !asdf.IsOnline() {
					Skip("skipping online test (set ONLINE=1 to run)")
				}
			})

			It("updates tools with latest versions", func() {
				content := "golang latest\nnodejs latest\nunknown 1.0.0\n"
				Expect(os.WriteFile(".tool-versions", []byte(content), asdf.CommonFilePermission)).To(Succeed())

				os.Args = []string{"cmd", "update-tool-versions"} //nolint:reassign // tests intentionally override os.Args

				err := cmdUpdateToolVersions()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("cmdGenerateToolSums", func() {
		var (
			originalArgs    []string
			originalCwd     string
			originalDataDir string
			tempDir         string
		)

		BeforeEach(func() {
			var err error
			originalArgs = os.Args
			originalCwd, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			originalDataDir = os.Getenv("ASDF_DATA_DIR")

			tempDir, err = os.MkdirTemp("", "generate-tool-sums-*")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tempDir)).To(Succeed())
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
			if originalDataDir == "" {
				os.Unsetenv("ASDF_DATA_DIR")
			} else {
				os.Setenv("ASDF_DATA_DIR", originalDataDir)
			}
			Expect(os.Chdir(originalCwd)).To(Succeed())
			os.RemoveAll(tempDir)
		})

		It("returns nil when no versions are defined", func() {
			os.Args = []string{"cmd", "generate-tool-sums"} //nolint:reassign // tests intentionally override os.Args

			Expect(os.WriteFile(".tool-versions", []byte("# none\n"), asdf.CommonFilePermission)).To(Succeed())

			Expect(cmdGenerateToolSums()).To(Succeed())
		})

		It("generates sums for installed tools", func() {
			toolVersions := "golang 1.21.0\nnodejs 20.0.0\nnightly nightly\n"
			Expect(os.WriteFile(".tool-versions", []byte(toolVersions), asdf.CommonFilePermission)).To(Succeed())

			asdfDir := filepath.Join(tempDir, "asdf-data")
			Expect(os.MkdirAll(asdfDir, asdf.CommonDirectoryPermission)).To(Succeed())
			os.Setenv("ASDF_DATA_DIR", asdfDir)

			for _, tc := range []struct {
				name    string
				version string
			}{
				{"golang", "1.21.0"},
				{"nodejs", "20.0.0"},
				// nightly should be skipped
			} {
				installDir := filepath.Join(asdfDir, "installs", tc.name, tc.version)
				Expect(os.MkdirAll(installDir, asdf.CommonDirectoryPermission)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(installDir, "bin"), []byte("x"), asdf.CommonFilePermission)).To(Succeed())
			}

			os.Args = []string{"cmd", "generate-tool-sums"} //nolint:reassign // tests intentionally override os.Args

			Expect(cmdGenerateToolSums()).To(Succeed())

			content, err := os.ReadFile(".tool-sums")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("golang 1.21.0"))
			Expect(string(content)).To(ContainSubstring("nodejs 20.0.0"))
		})
	})

	Describe("cmdHelpOverview", func() {
		DescribeTable("returns help overview for each plugin",
			func(pluginName string) {
				plugin, err := plugins.GetPlugin(pluginName)
				Expect(err).NotTo(HaveOccurred())

				err = cmdHelpOverview(plugin)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("golang", "golang"),
			Entry("python", "python"),
			Entry("nodejs", "nodejs"),
		)
	})

	Describe("cmdHelpDeps", func() {
		DescribeTable("returns help deps for each plugin",
			func(pluginName string) {
				plugin, err := plugins.GetPlugin(pluginName)
				Expect(err).NotTo(HaveOccurred())

				err = cmdHelpDeps(plugin)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("golang", "golang"),
			Entry("python", "python"),
			Entry("nodejs", "nodejs"),
		)
	})

	Describe("cmdHelpConfig", func() {
		DescribeTable("returns help config for each plugin",
			func(pluginName string) {
				plugin, err := plugins.GetPlugin(pluginName)
				Expect(err).NotTo(HaveOccurred())

				err = cmdHelpConfig(plugin)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("golang", "golang"),
			Entry("python", "python"),
			Entry("nodejs", "nodejs"),
		)
	})

	Describe("cmdHelpLinks", func() {
		DescribeTable("returns help links for each plugin",
			func(pluginName string) {
				plugin, err := plugins.GetPlugin(pluginName)
				Expect(err).NotTo(HaveOccurred())

				err = cmdHelpLinks(plugin)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("golang", "golang"),
			Entry("python", "python"),
			Entry("nodejs", "nodejs"),
		)
	})

	Describe("cmdLatestStable", func() {
		var originalArgs []string

		BeforeEach(func() {
			originalArgs = os.Args
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
		})

		When("running online tests", func() {
			BeforeEach(func() {
				if !asdf.IsOnline() {
					Skip("skipping online test (set ONLINE=1 to run)")
				}
			})

			It("returns latest stable Go version", func() {
				os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
				plugin, err := plugins.GetPlugin("golang")
				Expect(err).NotTo(HaveOccurred())

				err = cmdLatestStable(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("cmdListAll", func() {
		When("running online tests", func() {
			BeforeEach(func() {
				if !asdf.IsOnline() {
					Skip("skipping online test (set ONLINE=1 to run)")
				}
			})

			It("lists all Go versions", func() {
				plugin, err := plugins.GetPlugin("golang")
				Expect(err).NotTo(HaveOccurred())

				err = cmdListAll(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("lists all Python versions", func() {
				plugin, err := plugins.GetPlugin("python")
				Expect(err).NotTo(HaveOccurred())

				err = cmdListAll(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("lists all Node.js versions", func() {
				plugin, err := plugins.GetPlugin("nodejs")
				Expect(err).NotTo(HaveOccurred())

				err = cmdListAll(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("downloads Go to temp directory", func() {
				plugin, err := plugins.GetPlugin("golang")
				Expect(err).NotTo(HaveOccurred())

				tempDir, err := os.MkdirTemp("", "main-go-download-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				os.Setenv("ASDF_DOWNLOAD_PATH", tempDir)
				defer os.Unsetenv("ASDF_DOWNLOAD_PATH")
				os.Setenv("ASDF_INSTALL_VERSION", "1.21.0")
				defer os.Unsetenv("ASDF_INSTALL_VERSION")

				err = cmdDownload(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets latest stable Go version", func() {
				os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
				plugin, err := plugins.GetPlugin("golang")
				Expect(err).NotTo(HaveOccurred())

				err = cmdLatestStable(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets latest stable Python version", func() {
				os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
				plugin, err := plugins.GetPlugin("python")
				Expect(err).NotTo(HaveOccurred())

				err = cmdLatestStable(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets latest stable Node.js version", func() {
				os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
				plugin, err := plugins.GetPlugin("nodejs")
				Expect(err).NotTo(HaveOccurred())

				err = cmdLatestStable(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})

			It("installs Go to temp directory", func() {
				plugin, err := plugins.GetPlugin("golang")
				Expect(err).NotTo(HaveOccurred())

				tempDir, err := os.MkdirTemp("", "main-go-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				downloadPath := filepath.Join(tempDir, "download")
				installPath := filepath.Join(tempDir, "install")
				Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
				Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

				os.Setenv("ASDF_DOWNLOAD_PATH", downloadPath)
				defer os.Unsetenv("ASDF_DOWNLOAD_PATH")
				os.Setenv("ASDF_INSTALL_PATH", installPath)
				defer os.Unsetenv("ASDF_INSTALL_PATH")
				os.Setenv("ASDF_INSTALL_VERSION", "1.21.0")
				defer os.Unsetenv("ASDF_INSTALL_VERSION")

				err = cmdDownload(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())

				err = cmdInstall(context.Background(), plugin)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("cmdInstallPlugin", func() {
		var originalArgs []string
		var originalDataDir string
		var tempDir string

		BeforeEach(func() {
			originalArgs = os.Args
			originalDataDir = os.Getenv("ASDF_DATA_DIR")

			var err error
			tempDir, err = os.MkdirTemp("", "install-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())

			os.Setenv("ASDF_DATA_DIR", tempDir)
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
			if originalDataDir == "" {
				os.Unsetenv("ASDF_DATA_DIR")
			} else {
				os.Setenv("ASDF_DATA_DIR", originalDataDir)
			}
			os.RemoveAll(tempDir)
		})

		It("installs all plugins when no args provided", func() {
			os.Args = []string{"cmd", "install-plugin"} //nolint:reassign // tests intentionally override os.Args

			err := cmdInstallPlugin()
			Expect(err).NotTo(HaveOccurred())

			pluginsDir := filepath.Join(tempDir, "plugins")
			for _, plugin := range []string{"golang", "python", "nodejs"} {
				binDir := filepath.Join(pluginsDir, plugin, "bin")
				info, err := os.Stat(binDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(info.IsDir()).To(BeTrue())
			}
		})

		It("installs specific plugin when arg provided", func() {
			os.Args = []string{"cmd", "install-plugin", "golang"} //nolint:reassign // tests intentionally override os.Args

			err := cmdInstallPlugin()
			Expect(err).NotTo(HaveOccurred())

			pluginsDir := filepath.Join(tempDir, "plugins")
			binDir := filepath.Join(pluginsDir, "golang", "bin")
			info, err := os.Stat(binDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())

			_, err = os.Stat(filepath.Join(pluginsDir, "python"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("installs multiple specific plugins", func() {
			os.Args = []string{"cmd", "install-plugin", "golang", "nodejs"} //nolint:reassign // tests intentionally override os.Args

			err := cmdInstallPlugin()
			Expect(err).NotTo(HaveOccurred())

			pluginsDir := filepath.Join(tempDir, "plugins")
			for _, plugin := range []string{"golang", "nodejs"} {
				binDir := filepath.Join(pluginsDir, plugin, "bin")
				info, err := os.Stat(binDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(info.IsDir()).To(BeTrue())
			}

			_, err = os.Stat(filepath.Join(pluginsDir, "python"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("returns error for unknown plugin", func() {
			os.Args = []string{"cmd", "install-plugin", "unknown"} //nolint:reassign // tests intentionally override os.Args

			err := cmdInstallPlugin()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("run with install-plugin", func() {
		var originalArgs []string
		var originalDataDir string
		var tempDir string

		BeforeEach(func() {
			originalArgs = os.Args
			originalDataDir = os.Getenv("ASDF_DATA_DIR")

			var err error
			tempDir, err = os.MkdirTemp("", "run-install-plugin-test-*")
			Expect(err).NotTo(HaveOccurred())

			os.Setenv("ASDF_DATA_DIR", tempDir)
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
			if originalDataDir == "" {
				os.Unsetenv("ASDF_DATA_DIR")
			} else {
				os.Setenv("ASDF_DATA_DIR", originalDataDir)
			}
			os.RemoveAll(tempDir)
		})

		It("handles install-plugin command via run", func() {
			os.Args = []string{"cmd", "install-plugin", "golang"} //nolint:reassign // tests intentionally override os.Args

			err := run()
			Expect(err).NotTo(HaveOccurred())

			pluginsDir := filepath.Join(tempDir, "plugins")
			binDir := filepath.Join(pluginsDir, "golang", "bin")
			info, err := os.Stat(binDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})
	})

	Describe("cmdListAll with mock plugin", func() {
		It("lists all versions from mock plugin", func() {
			mock := &mockPlugin{
				name:     "mock",
				versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			}

			err := cmdListAll(context.Background(), mock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when plugin fails", func() {
			mock := &mockPlugin{
				name:       "mock",
				listAllErr: true,
			}

			err := cmdListAll(context.Background(), mock)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("cmdLatestStable with mock plugin", func() {
		var originalArgs []string

		BeforeEach(func() {
			originalArgs = os.Args
		})

		AfterEach(func() {
			os.Args = originalArgs //nolint:reassign // tests intentionally override os.Args
		})

		It("returns latest stable version", func() {
			os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
			mock := &mockPlugin{
				name:         "mock",
				latestStable: "3.0.0",
			}

			err := cmdLatestStable(context.Background(), mock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns latest stable with query", func() {
			os.Args = []string{"cmd", "latest-stable", "2"} //nolint:reassign // tests intentionally override os.Args
			mock := &mockPlugin{
				name:         "mock",
				latestStable: "2.5.0",
			}

			err := cmdLatestStable(context.Background(), mock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when plugin fails", func() {
			os.Args = []string{"cmd", "latest-stable"} //nolint:reassign // tests intentionally override os.Args
			mock := &mockPlugin{
				name:            "mock",
				latestStableErr: true,
			}

			err := cmdLatestStable(context.Background(), mock)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("cmdDownload with mock plugin", func() {
		var tempDir string
		var originalVersion, originalPath, originalCwd string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "download-test-*")
			Expect(err).NotTo(HaveOccurred())

			originalVersion = os.Getenv("ASDF_INSTALL_VERSION")
			originalPath = os.Getenv("ASDF_DOWNLOAD_PATH")

			originalCwd, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tempDir)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Chdir(originalCwd)).To(Succeed())
			os.RemoveAll(tempDir)
			os.Setenv("ASDF_INSTALL_VERSION", originalVersion)
			os.Setenv("ASDF_DOWNLOAD_PATH", originalPath)
		})

		It("downloads successfully", func() {
			os.Setenv("ASDF_INSTALL_VERSION", "1.0.0")
			os.Setenv("ASDF_DOWNLOAD_PATH", tempDir)

			mock := &mockPlugin{name: "mock"}

			err := cmdDownload(context.Background(), mock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when plugin download fails", func() {
			os.Setenv("ASDF_INSTALL_VERSION", "1.0.0")
			os.Setenv("ASDF_DOWNLOAD_PATH", tempDir)

			mock := &mockPlugin{
				name:        "mock",
				downloadErr: true,
			}

			err := cmdDownload(context.Background(), mock)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
		})
	})

	Describe("cmdInstall with mock plugin", func() {
		var tempDir string
		var originalVersion, originalInstallPath, originalDownloadPath string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "install-test-*")
			Expect(err).NotTo(HaveOccurred())

			originalVersion = os.Getenv("ASDF_INSTALL_VERSION")
			originalInstallPath = os.Getenv("ASDF_INSTALL_PATH")
			originalDownloadPath = os.Getenv("ASDF_DOWNLOAD_PATH")
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
			os.Setenv("ASDF_INSTALL_VERSION", originalVersion)
			os.Setenv("ASDF_INSTALL_PATH", originalInstallPath)
			os.Setenv("ASDF_DOWNLOAD_PATH", originalDownloadPath)
		})

		It("installs successfully with download path", func() {
			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			os.Setenv("ASDF_INSTALL_VERSION", "1.0.0")
			os.Setenv("ASDF_INSTALL_PATH", installPath)
			os.Setenv("ASDF_DOWNLOAD_PATH", downloadPath)

			mock := &mockPlugin{name: "mock"}

			err := cmdInstall(context.Background(), mock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("installs successfully without download path", func() {
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			os.Setenv("ASDF_INSTALL_VERSION", "1.0.0")
			os.Setenv("ASDF_INSTALL_PATH", installPath)
			os.Unsetenv("ASDF_DOWNLOAD_PATH")

			mock := &mockPlugin{name: "mock"}

			err := cmdInstall(context.Background(), mock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when plugin install fails without download path", func() {
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			os.Setenv("ASDF_INSTALL_VERSION", "1.0.0")
			os.Setenv("ASDF_INSTALL_PATH", installPath)
			os.Unsetenv("ASDF_DOWNLOAD_PATH")

			mock := &mockPlugin{
				name:       "mock",
				installErr: true,
			}

			err := cmdInstall(context.Background(), mock)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when plugin install fails", func() {
			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(installPath, asdf.CommonDirectoryPermission)).To(Succeed())

			os.Setenv("ASDF_INSTALL_VERSION", "1.0.0")
			os.Setenv("ASDF_INSTALL_PATH", installPath)
			os.Setenv("ASDF_DOWNLOAD_PATH", downloadPath)

			mock := &mockPlugin{
				name:       "mock",
				installErr: true,
			}

			err := cmdInstall(context.Background(), mock)
			Expect(err).To(HaveOccurred())
		})
	})
})

// mockPlugin implements asdf.Plugin for testing.
type mockPlugin struct {
	name            string
	latestStable    string
	versions        []string
	listAllErr      bool
	downloadErr     bool
	installErr      bool
	latestStableErr bool
}

// Name returns the plugin name.
func (m *mockPlugin) Name() string { return m.name }

// ListAll returns mock versions or an error if configured.
func (m *mockPlugin) ListAll(_ context.Context) ([]string, error) {
	if m.listAllErr {
		return nil, context.DeadlineExceeded
	}

	return m.versions, nil
}

// Download simulates downloading a version.
func (m *mockPlugin) Download(_ context.Context, _, _ string) error {
	if m.downloadErr {
		return context.DeadlineExceeded
	}

	return nil
}

// Install simulates installing a version.
func (m *mockPlugin) Install(_ context.Context, _, _, _ string) error {
	if m.installErr {
		return context.DeadlineExceeded
	}

	return nil
}

// Uninstall simulates uninstalling a version.
func (*mockPlugin) Uninstall(_ context.Context, _ string) error {
	return nil
}

// ListBinPaths returns the mock binary paths.
func (*mockPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns mock environment variables.
func (*mockPlugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns mock legacy filenames.
func (*mockPlugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a mock legacy file.
func (*mockPlugin) ParseLegacyFile(_ string) (string, error) {
	return "", nil
}

// LatestStable returns the mock latest stable version.
func (m *mockPlugin) LatestStable(_ context.Context, _ string) (string, error) {
	if m.latestStableErr {
		return "", context.DeadlineExceeded
	}

	return m.latestStable, nil
}

// Help returns mock help information.
func (*mockPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{}
}

var _ = Describe("Tool Version Functions", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "tool-version-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("parseToolVersions", func() {
		It("parses valid .tool-versions file", func() {
			content := `golang 1.21.0
python 3.12.0
# comment line
nodejs 20.0.0
`
			filePath := filepath.Join(tempDir, ".tool-versions")
			err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			tools, err := parseToolVersions(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(tools).To(HaveLen(3))
			Expect(tools["golang"]).To(Equal("1.21.0"))
			Expect(tools["python"]).To(Equal("3.12.0"))
			Expect(tools["nodejs"]).To(Equal("20.0.0"))
		})

		It("returns error for non-existent file", func() {
			_, err := parseToolVersions("/non/existent/file")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("writeToolVersions", func() {
		It("writes tool versions to file", func() {
			filePath := filepath.Join(tempDir, ".tool-versions")
			tools := map[string]string{
				"golang": "1.21.0",
				"python": "3.12.0",
			}

			err := writeToolVersions(filePath, tools)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("golang 1.21.0"))
			Expect(string(content)).To(ContainSubstring("python 3.12.0"))
		})
	})

	Describe("parseToolSums", func() {
		It("parses valid .tool-sums file", func() {
			content := `golang 1.21.0 abc123
python 3.12.0 def456
`
			filePath := filepath.Join(tempDir, ".tool-sums")
			err := os.WriteFile(filePath, []byte(content), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			sums, err := parseToolSums(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(sums).To(HaveLen(2))
			Expect(sums["golang:1.21.0"]).To(Equal("abc123"))
			Expect(sums["python:3.12.0"]).To(Equal("def456"))
		})

		It("returns empty map for non-existent file", func() {
			sums, err := parseToolSums("/non/existent/file")
			Expect(err).NotTo(HaveOccurred())
			Expect(sums).To(BeEmpty())
		})
	})

	Describe("writeToolSums", func() {
		It("writes tool sums to file", func() {
			filePath := filepath.Join(tempDir, ".tool-sums")
			sums := map[string]string{
				"golang:1.21.0": "abc123",
				"python:3.12.0": "def456",
			}

			err := writeToolSums(filePath, sums)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("golang 1.21.0 abc123"))
			Expect(string(content)).To(ContainSubstring("python 3.12.0 def456"))
		})
	})

	Describe("calculateFileHash", func() {
		It("calculates hash of a file", func() {
			filePath := filepath.Join(tempDir, "testfile")
			err := os.WriteFile(filePath, []byte("test content"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			hash, err := calculateFileHash(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(hash).NotTo(BeEmpty())
			Expect(hash).To(HavePrefix("sha256:"))
		})

		It("returns error for non-existent file", func() {
			_, err := calculateFileHash("/non/existent/file")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("calculateDirHash", func() {
		It("calculates hash of a directory", func() {
			dirPath := filepath.Join(tempDir, "testdir")
			Expect(os.MkdirAll(dirPath, asdf.CommonDirectoryPermission)).To(Succeed())

			err := os.WriteFile(filepath.Join(dirPath, "file1.txt"), []byte("content1"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(dirPath, "file2.txt"), []byte("content2"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			hash, err := calculateDirHash(dirPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(hash).NotTo(BeEmpty())
		})

		It("returns empty hash for non-existent directory", func() {
			hash, err := calculateDirHash("/non/existent/dir")
			Expect(err).NotTo(HaveOccurred())

			Expect(hash).To(HavePrefix("sha256:"))
		})
	})
})
