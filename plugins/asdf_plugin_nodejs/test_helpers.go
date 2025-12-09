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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
)

type (
	// VersionInfo represents Node.js version information.
	VersionInfo struct {
		Version string `json:"version"`
		LTS     any    `json:"lts"`
		Date    string `json:"date"`
	}

	// nodeTestFixture provides plugin instances for testing.
	nodeTestFixture struct {
		plugin   *Plugin
		server   *mock.Server
		versions []VersionInfo
	}
)

// newNodeTestFixture creates a test fixture for the plugin.
// In ONLINE mode, it uses the real plugin; otherwise it uses a mock fixture.
func newNodeTestFixture() *nodeTestFixture {
	fixture := &nodeTestFixture{}

	if asdf.IsOnline() {
		fixture.plugin = New()
		return fixture
	}

	return newNodeMockFixture()
}

// newNodeMockFixture creates a mock-only test fixture for the plugin.
// It always uses the mock server and is independent of ONLINE mode.
func newNodeMockFixture() *nodeTestFixture {
	fixture := &nodeTestFixture{}

	fixture.server = mock.NewServer("nodejs", "node")
	fixture.plugin = NewWithURLs(fixture.server.URL()+"/dist/index.json", fixture.server.URL()+"/dist/")

	return fixture
}

// Close shuts down the mock server if it was created.
func (fixture *nodeTestFixture) Close() {
	if fixture.server != nil {
		fixture.server.Close()
	}
}

// SetupVersion registers a mocked Node.js release and associated metadata
// (archive, checksum, index entry) for the given version, platform, arch, and
// LTS value on the test server.
func (fixture *nodeTestFixture) SetupVersion(version, platform, arch string, lts any) {
	archive, err := buildNodeArchive(version, platform, arch)
	Expect(err).NotTo(HaveOccurred())

	archivePath := fmt.Sprintf("/dist/v%s/node-v%s-%s-%s.tar.gz", version, version, platform, arch)
	fixture.server.RegisterFile(archivePath, archive)

	checksum := sha256sum(archive)
	filename := fmt.Sprintf("node-v%s-%s-%s.tar.gz", version, platform, arch)
	shasumContent := fmt.Sprintf("%s  %s\n", checksum, filename)
	shasumPath := fmt.Sprintf("/dist/v%s/SHASUMS256.txt", version)
	fixture.server.RegisterText(shasumPath, shasumContent)

	fixture.versions = append(fixture.versions, VersionInfo{
		Version: "v" + version,
		LTS:     lts,
		Date:    "2024-01-01",
	})
	fixture.server.RegisterJSON("/dist/index.json", fixture.versions)
}

// buildNodeArchive constructs a minimal tar.gz Node.js distribution archive
// with basic bin stubs for node, npm, and npx for use in tests.
func buildNodeArchive(version, platform, arch string) ([]byte, error) {
	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	dirName := fmt.Sprintf("node-v%s-%s-%s", version, platform, arch)
	files := map[string]string{
		dirName + "/bin/node": "#!/bin/bash\necho v" + version,
		dirName + "/bin/npm":  "#!/bin/bash\necho npm",
		dirName + "/bin/npx":  "#!/bin/bash\necho npx",
	}

	for name, content := range files {
		mode := int64(asdf.TarFilePermission)
		if strings.Contains(name, "/bin/") {
			mode = int64(asdf.CommonDirectoryPermission)
		}

		hdr := &tar.Header{
			Name: name,
			Mode: mode,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		if _, err := tw.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// sha256sum returns the SHA-256 checksum of the given data as a hex string.
func sha256sum(data []byte) string {
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}
