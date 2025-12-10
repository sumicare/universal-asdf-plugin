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

package asdf_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var _ = Describe("BinaryPlugin", func() {
	var (
		server *mock.Server
		plugin *asdf.BinaryPlugin
		config asdf.BinaryPluginConfig
	)

	BeforeEach(func() {
		server = mock.NewServer("owner", "repo")
		config = asdf.BinaryPluginConfig{
			Name:       "test-tool",
			RepoOwner:  "owner",
			RepoName:   "repo",
			BinaryName: "test-tool",
		}
		plugin = asdf.NewBinaryPlugin(&config)
		plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("ListAll", func() {
		It("lists versions from tags", func() {
			server.RegisterTag("v1.0.0")
			server.RegisterTag("v1.1.0")
			server.RegisterTag("v2.0.0")

			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.0.0", "1.1.0", "2.0.0"))
		})

		It("handles tags without prefix if configured", func() {
			config.VersionPrefix = ""
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			server.RegisterTag("1.0.0")
			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.0.0"))
		})

		It("uses tags when UseReleases is false", func() {
			useReleases := false
			config.UseReleases = &useReleases
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			server.RegisterTag("v1.0.0")
			server.RegisterTag("v1.1.0")

			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.0.0", "1.1.0"))
		})

		It("applies VersionFilter regex when provided", func() {
			config.VersionFilter = `^1`
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			server.RegisterTag("v1.0.0")
			server.RegisterTag("v1.1.0")
			server.RegisterTag("v2.0.0")

			versions, err := plugin.ListAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(ContainElements("1.0.0", "1.1.0"))
			Expect(versions).NotTo(ContainElement("2.0.0"))
		})

		It("returns error for invalid VersionFilter regex", func() {
			config.VersionFilter = "["
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			server.RegisterTag("v1.0.0")

			_, err := plugin.ListAll(context.Background())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download", func() {
		It("downloads the correct file", func() {
			platform, err := asdf.GetPlatform()
			Expect(err).NotTo(HaveOccurred())
			arch, err := asdf.GetArch()
			Expect(err).NotTo(HaveOccurred())

			filename := fmt.Sprintf("test-tool-%s-%s", platform, arch)
			path := "/owner/repo/releases/download/v1.0.0/" + filename
			server.RegisterDownload(path)

			config.DownloadURLTemplate = server.URL() + "/{{.RepoOwner}}/{{.RepoName}}/releases/download/v{{.Version}}/{{.FileName}}"
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			tempDir, err := os.MkdirTemp("", "generic-download-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).NotTo(HaveOccurred())

			stat, err := os.Stat(filepath.Join(tempDir, filename))
			Expect(err).NotTo(HaveOccurred())
			Expect(stat.Mode()&asdf.ExecutablePermissionMask).NotTo(BeZero(), "should be executable")
		})

		It("supports custom filename templates", func() {
			config.FileNameTemplate = "custom-{{.Version}}-{{.Platform}}"
			config.DownloadURLTemplate = server.URL() + "/{{.RepoOwner}}/{{.RepoName}}/releases/download/v{{.Version}}/{{.FileName}}"
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			platform, err := asdf.GetPlatform()
			Expect(err).NotTo(HaveOccurred())
			filename := "custom-1.0.0-" + platform
			path := "/owner/repo/releases/download/v1.0.0/" + filename
			server.RegisterDownload(path)

			tempDir, err := os.MkdirTemp("", "generic-download-custom-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(tempDir, filename)).To(BeAnExistingFile())
		})

		It("returns error for unsupported architecture", func() {
			originalArch := os.Getenv("ASDF_OVERWRITE_ARCH")
			defer os.Setenv("ASDF_OVERWRITE_ARCH", originalArch)

			config.ArchMap = map[string]string{
				"amd64": "amd64",
			}
			plugin = asdf.NewBinaryPlugin(&config)
			plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

			os.Setenv("ASDF_OVERWRITE_ARCH", "arm64")

			tempDir, err := os.MkdirTemp("", "generic-download-unsupported-arch-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported architecture"))
		})

		It("uses cached download when file already exists", func() {
			platform, err := asdf.GetPlatform()
			Expect(err).NotTo(HaveOccurred())
			arch, err := asdf.GetArch()
			Expect(err).NotTo(HaveOccurred())

			filename := fmt.Sprintf("test-tool-%s-%s", platform, arch)

			tempDir, err := os.MkdirTemp("", "generic-download-cache-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			binaryPath := filepath.Join(tempDir, filename)
			// create a file larger than 1KiB so the cache branch is taken
			payload := bytes.Repeat([]byte{'x'}, 2048)
			err = os.WriteFile(binaryPath, payload, asdf.CommonExecutablePermission)
			Expect(err).NotTo(HaveOccurred())

			// No download registration required; Download should return early.
			err = plugin.Download(context.Background(), "1.0.0", tempDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Install", func() {
		It("installs binary from file", func() {
			tempDir, err := os.MkdirTemp("", "generic-install-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())

			err = os.WriteFile(filepath.Join(downloadPath, "some-binary"), []byte("content"), asdf.CommonDirectoryPermission)
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(installPath, "bin", "test-tool")).To(BeAnExistingFile())
		})

		DescribeTable("installs binary from archive",
			func(archiveType, archiveFile string, createArchive func(string, map[string]string)) {
				config.ArchiveType = archiveType
				plugin = asdf.NewBinaryPlugin(&config)
				plugin.WithGithubClient(github.NewClientWithHTTP(server.Client(), server.URL()))

				tempDir, err := os.MkdirTemp("", "generic-install-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				downloadPath := filepath.Join(tempDir, "download")
				installPath := filepath.Join(tempDir, "install")
				Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())

				archivePath := filepath.Join(downloadPath, archiveFile)
				createArchive(archivePath, map[string]string{"test-tool": "binary content"})

				err = plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(filepath.Join(installPath, "bin", "test-tool")).To(BeAnExistingFile())
			},
			Entry("tar.gz", "tar.gz", "archive.tar.gz", asdf.CreateTestTarGz),
			Entry("tar.xz", "tar.xz", "archive.tar.xz", asdf.CreateTestTarXz),
			Entry("zip", "zip", "archive.zip", asdf.CreateTestZip),
		)

		It("installs binary from gz archive", func() {
			config.ArchiveType = "gz"
			plugin = asdf.NewBinaryPlugin(&config)

			tempDir, err := os.MkdirTemp("", "generic-install-gz-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())

			archivePath := filepath.Join(downloadPath, "test-tool.gz")
			asdf.CreateTestGz(archivePath, "binary content")

			err = plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(installPath, "bin", "test-tool")).To(BeAnExistingFile())
		})

		It("returns error when no binary is present in download directory", func() {
			tempDir, err := os.MkdirTemp("", "generic-install-nobinary-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			downloadPath := filepath.Join(tempDir, "download")
			installPath := filepath.Join(tempDir, "install")
			Expect(os.MkdirAll(downloadPath, asdf.CommonDirectoryPermission)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(downloadPath, "subdir"), asdf.CommonDirectoryPermission)).To(Succeed())

			err = plugin.Install(context.Background(), "1.0.0", downloadPath, installPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no binary found"))
		})
	})

	Describe("WithGithubClient", func() {
		It("sets the github client", func() {
			client := github.NewClient()
			p := plugin.WithGithubClient(client)
			Expect(p.Github).To(Equal(client))
		})
	})

	Describe("Uninstall", func() {
		It("removes the installation directory", func() {
			tempDir, err := os.MkdirTemp("", "generic-uninstall-*")
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(tempDir, "file"), []byte("content"), asdf.CommonFilePermission)
			Expect(err).NotTo(HaveOccurred())

			err = plugin.Uninstall(context.Background(), tempDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tempDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("Misc Methods", func() {
		It("returns correct Name", func() {
			Expect(plugin.Name()).To(Equal("test-tool"))
		})

		It("returns correct ListBinPaths", func() {
			Expect(plugin.ListBinPaths()).To(Equal("bin"))
		})

		It("returns empty ExecEnv", func() {
			env := plugin.ExecEnv("/some/path")
			Expect(env).To(BeEmpty())
		})

		It("returns empty ListLegacyFilenames", func() {
			files := plugin.ListLegacyFilenames()
			Expect(files).To(BeEmpty())
		})

		It("ParseLegacyFile reads content", func() {
			tempFile, err := os.CreateTemp("", "legacy-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString("1.2.3\n")
			Expect(err).NotTo(HaveOccurred())
			tempFile.Close()

			version, err := plugin.ParseLegacyFile(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.3"))
		})

		It("Help returns valid help info", func() {
			config.HelpDescription = "Test Description"
			config.HelpLink = "http://example.com"
			plugin = asdf.NewBinaryPlugin(&config)

			help := plugin.Help()
			Expect(help.Overview).To(ContainSubstring("test-tool - Test Description"))
			Expect(help.Links).To(ContainSubstring("http://example.com"))
		})
	})

	Describe("LatestStable", func() {
		It("returns latest stable version", func() {
			server.RegisterTag("v1.0.0")
			server.RegisterTag("v1.1.0")
			server.RegisterTag("v1.1.0")

			version, err := plugin.LatestStable(context.Background(), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.1.0"))
		})

		It("filters by query", func() {
			server.RegisterTag("v1.0.0")
			server.RegisterTag("v1.1.0")
			server.RegisterTag("v2.0.0")

			version, err := plugin.LatestStable(context.Background(), "1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.0.0"))
		})
	})
})
