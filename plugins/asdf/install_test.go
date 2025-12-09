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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// PluginInstaller unit tests exercise the high-level installer helper
// without invoking a real asdf installation.
var _ = Describe("PluginInstaller", func() {
	var (
		tmpDir     string
		pluginsDir string
		execPath   string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "asdf-install-test-*")
		Expect(err).NotTo(HaveOccurred())

		pluginsDir = filepath.Join(tmpDir, "plugins")

		execPath = filepath.Join(tmpDir, "test-plugin")
		err = os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), CommonDirectoryPermission)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("NewPluginInstaller", func() {
		It("creates installer with resolved exec path", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(installer.ExecPath).To(Equal(execPath))
			Expect(installer.PluginsDir).To(Equal(pluginsDir))
		})

		It("uses GetPluginsDir when pluginsDir is empty", func() {
			original := os.Getenv("ASDF_DATA_DIR")
			defer os.Setenv("ASDF_DATA_DIR", original)

			os.Setenv("ASDF_DATA_DIR", tmpDir)

			installer, err := NewPluginInstaller(execPath, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(installer.PluginsDir).To(Equal(filepath.Join(tmpDir, "plugins")))
		})
	})

	Describe("GetPluginsDir", func() {
		var originalDataDir string

		BeforeEach(func() {
			originalDataDir = os.Getenv("ASDF_DATA_DIR")
		})

		AfterEach(func() {
			if originalDataDir == "" {
				os.Unsetenv("ASDF_DATA_DIR")
			} else {
				os.Setenv("ASDF_DATA_DIR", originalDataDir)
			}
		})

		It("uses ASDF_DATA_DIR when set", func() {
			os.Setenv("ASDF_DATA_DIR", "/custom/asdf")
			Expect(GetPluginsDir()).To(Equal("/custom/asdf/plugins"))
		})

		It("falls back to ~/.asdf/plugins", func() {
			os.Unsetenv("ASDF_DATA_DIR")
			home, err := os.UserHomeDir()
			Expect(err).NotTo(HaveOccurred())
			Expect(GetPluginsDir()).To(Equal(filepath.Join(home, ".asdf", "plugins")))
		})
	})

	Describe("Install", func() {
		It("creates plugin directory structure", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())

			binDir := filepath.Join(pluginsDir, "golang", "bin")
			info, err := os.Stat(binDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})

		It("creates all required wrapper scripts", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())

			expectedScripts := []string{
				"list-all",
				"download",
				"install",
				"uninstall",
				"list-bin-paths",
				"exec-env",
				"latest-stable",
				"list-legacy-filenames",
				"parse-legacy-file",
				"help.overview",
				"help.deps",
				"help.config",
				"help.links",
			}

			binDir := filepath.Join(pluginsDir, "golang", "bin")
			for _, script := range expectedScripts {
				scriptPath := filepath.Join(binDir, script)
				info, err := os.Stat(scriptPath)
				Expect(err).NotTo(HaveOccurred(), "script %s should exist", script)
				Expect(info.Mode().Perm()&ExecutablePermissionMask).NotTo(BeZero(), "script %s should be executable", script)
			}
		})

		It("generates correct wrapper script content", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())

			scriptPath := filepath.Join(pluginsDir, "golang", "bin", "list-all")
			content, err := os.ReadFile(scriptPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("#!/usr/bin/env bash"))
			Expect(string(content)).To(ContainSubstring("set -euo pipefail"))
			Expect(string(content)).To(ContainSubstring(`ASDF_PLUGIN_NAME="golang"`))
			Expect(string(content)).To(ContainSubstring(execPath))
			Expect(string(content)).To(ContainSubstring(`"list-all"`))
		})
	})

	Describe("InstallAll", func() {
		It("installs all available plugins", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			installed, err := installer.InstallAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(installed).To(ConsistOf("golang", "python", "nodejs"))

			for _, plugin := range installed {
				binDir := filepath.Join(pluginsDir, plugin, "bin")
				info, err := os.Stat(binDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(info.IsDir()).To(BeTrue())
			}
		})
	})

	Describe("Uninstall", func() {
		It("removes plugin directory", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())

			err = installer.Uninstall("golang")
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(pluginsDir, "golang"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("IsInstalled", func() {
		It("returns true for installed plugin", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())

			Expect(installer.IsInstalled("golang")).To(BeTrue())
		})

		It("returns false for non-installed plugin", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(installer.IsInstalled("golang")).To(BeFalse())
		})
	})

	Describe("GetInstalledPlugins", func() {
		It("returns list of installed plugins", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())
			err = installer.Install("nodejs")
			Expect(err).NotTo(HaveOccurred())

			plugins, err := installer.GetInstalledPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).To(ConsistOf("golang", "nodejs"))
		})

		It("returns empty list when no plugins installed", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			plugins, err := installer.GetInstalledPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).To(BeEmpty())
		})
	})

	Describe("AvailablePlugins", func() {
		It("returns all available plugin names", func() {
			plugins := AvailablePlugins()
			Expect(plugins).To(ContainElements("golang", "python", "nodejs", "jq", "kubectl", "terraform"))
			Expect(len(plugins)).To(BeNumerically(">=", 40))
		})
	})

	Describe("NewPluginInstaller error cases", func() {
		It("returns error for non-existent executable", func() {
			_, err := NewPluginInstaller("/nonexistent/path/to/binary", pluginsDir)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Install error cases", func() {
		It("returns error when bin directory cannot be created", func() {
			installer, err := NewPluginInstaller(execPath, "/nonexistent/readonly/path")
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetInstalledPlugins edge cases", func() {
		It("ignores directories without bin subdirectory", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = os.MkdirAll(filepath.Join(pluginsDir, "fake-plugin"), CommonDirectoryPermission)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Install("golang")
			Expect(err).NotTo(HaveOccurred())

			plugins, err := installer.GetInstalledPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).To(ConsistOf("golang"))
			Expect(plugins).NotTo(ContainElement("fake-plugin"))
		})
	})

	Describe("InstallAll error cases", func() {
		It("returns error and partial list when install fails", func() {
			installer, err := NewPluginInstaller(execPath, "/nonexistent/readonly/path")
			Expect(err).NotTo(HaveOccurred())

			installed, err := installer.InstallAll()
			Expect(err).To(HaveOccurred())
			Expect(installed).To(BeEmpty())
		})
	})

	Describe("Uninstall edge cases", func() {
		It("succeeds even if plugin doesn't exist", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			err = installer.Uninstall("nonexistent")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("IsInstalled edge cases", func() {
		It("returns false when bin exists but is a file", func() {
			installer, err := NewPluginInstaller(execPath, pluginsDir)
			Expect(err).NotTo(HaveOccurred())

			pluginDir := filepath.Join(pluginsDir, "badplugin")
			err = os.MkdirAll(pluginDir, CommonDirectoryPermission)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(pluginDir, "bin"), []byte("not a dir"), CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			Expect(installer.IsInstalled("badplugin")).To(BeFalse())
		})
	})
})

// PluginInstaller with live asdf runs slower, serialised tests that
// shell out to the real asdf CLI to verify end-to-end compatibility.
var _ = Describe("PluginInstaller with live asdf", Serial, func() {
	var (
		tmpDir     string
		pluginsDir string
		installer  *PluginInstaller
	)

	BeforeEach(func() {
		if _, err := exec.LookPath("asdf"); err != nil {
			Skip("asdf not available in PATH")
		}

		_, thisFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		tmpDir = filepath.Join(filepath.Dir(thisFile), ".tmp", "live-asdf-test")

		os.RemoveAll(tmpDir)

		var err error
		err = os.MkdirAll(tmpDir, CommonDirectoryPermission)
		Expect(err).NotTo(HaveOccurred())

		pluginsDir = filepath.Join(tmpDir, "plugins")

		buildDir := filepath.Join(tmpDir, "build")
		err = os.MkdirAll(buildDir, CommonDirectoryPermission)
		Expect(err).NotTo(HaveOccurred())

		projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

		execPath := filepath.Join(buildDir, "universal-asdf-plugin")
		cmd := exec.Command("go", "build", "-o", execPath, ".")
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "build failed: %s", string(output))

		installer, err = NewPluginInstaller(execPath, pluginsDir)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("installs plugin that asdf can recognize", func() {
		err := installer.Install("golang")
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("asdf", "plugin", "list")
		cmd.Env = append(os.Environ(), "ASDF_DATA_DIR="+tmpDir)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "asdf plugin list failed: %s", string(output))
		Expect(string(output)).To(ContainSubstring("golang"))
	})

	It("installed plugin can list versions via asdf", func() {
		if !IsOnline() {
			Skip("ONLINE=1 not set, skipping network test")
		}

		err := installer.Install("golang")
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("asdf", "list", "all", "golang")
		cmd.Env = append(os.Environ(), "ASDF_DATA_DIR="+tmpDir)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "asdf list all failed: %s", string(output))

		versions := strings.Fields(string(output))
		Expect(len(versions)).To(BeNumerically(">", 10))
		Expect(versions).To(ContainElement("1.21.0"))
	})

	It("installed plugin can get latest stable via asdf", func() {
		if !IsOnline() {
			Skip("ONLINE=1 not set, skipping network test")
		}

		err := installer.Install("golang")
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("asdf", "latest", "golang")
		cmd.Env = append(os.Environ(), "ASDF_DATA_DIR="+tmpDir)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "asdf latest failed: %s", string(output))

		version := strings.TrimSpace(string(output))
		Expect(version).To(MatchRegexp(`^\d+\.\d+\.\d+$`))
	})
})
