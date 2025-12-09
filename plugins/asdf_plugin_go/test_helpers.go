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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// goTestFixture provides plugin instances for testing.
type goTestFixture struct {
	plugin       *Plugin
	server       *mock.Server
	testdataPath string
}

// newGoMockFixture constructs a mock-only test fixture for the Go plugin.
// It always uses the mock server and is independent of ONLINE mode.
func newGoMockFixture() *goTestFixture {
	fixture := &goTestFixture{}

	_, file, _, _ := runtime.Caller(0)

	fixture.testdataPath = filepath.Join(filepath.Dir(file), "testdata")

	fixture.server = mock.NewServer("golang", "go")

	githubClient := github.NewClientWithHTTP(&http.Client{}, fixture.server.URL())

	fixture.plugin = NewWithURLs(fixture.server.URL()+"/go", githubClient)

	return fixture
}

// Close shuts down the mock server if one was created.
func (fixture *goTestFixture) Close() {
	if fixture.server != nil {
		fixture.server.Close()
	}
}

// SetupVersion registers a mock download, checksum, and tag for the given version/platform/arch.
func (fixture *goTestFixture) SetupVersion(version, platform, arch string) {
	archive, err := buildGoArchive(version)
	Expect(err).NotTo(HaveOccurred())

	filename := fmt.Sprintf("go%s.%s-%s.tar.gz", version, platform, arch)
	path := "/go/" + filename

	fixture.server.RegisterFile(path, archive)

	checksum := sha256sum(archive)
	fixture.server.RegisterText(path+".sha256", checksum)

	fixture.server.RegisterTag("go" + version)
}

// SetupTags registers the given tags with the mock server for mock testing.
func (fixture *goTestFixture) SetupTags(tags []string) {
	for _, tag := range tags {
		fixture.server.RegisterTag(tag)
	}
}

// buildGoArchive builds an in-memory Go distribution tar.gz archive for the
// given version, used by tests to simulate official download artifacts.
func buildGoArchive(version string) ([]byte, error) {
	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	files := map[string]string{
		"go/VERSION":   "go" + version,
		"go/bin/go":    "#!/bin/bash\necho go version go" + version,
		"go/bin/gofmt": "#!/bin/bash\necho gofmt",
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

// sha256sum returns the hex-encoded SHA-256 checksum of the given data.
func sha256sum(data []byte) string {
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}

// SetupTagsFromGoldie reads versions from goldie test data and registers them as
// Git tags using the "go" prefix (e.g. go1.21.0). This matches the upstream
// Go repository tag format so that parseGoTags can correctly extract versions.
func (fixture *goTestFixture) SetupTagsFromGoldie() error {
	versions, err := testutil.ReadGoldieVersions(fixture.testdataPath, "go_list_all.golden")
	if err != nil {
		return err
	}

	tags := make([]string, 0, len(versions))
	for i := range versions {
		tags = append(tags, "go"+versions[i])
	}

	fixture.SetupTags(tags)

	return nil
}

// GoldieVersions returns the versions from the goldie test data.
func (fixture *goTestFixture) GoldieVersions() ([]string, error) {
	return testutil.ReadGoldieVersions(fixture.testdataPath, "go_list_all.golden")
}

// GoldieLatest returns the latest version from the goldie test data.
func (fixture *goTestFixture) GoldieLatest() (string, error) {
	return testutil.ReadGoldieLatest(fixture.testdataPath, "go_latest_stable.golden")
}

// GoldieFilterPattern returns a filter pattern based on goldie versions.
func (fixture *goTestFixture) GoldieFilterPattern() (string, error) {
	versions, err := fixture.GoldieVersions()
	if err != nil {
		return "", err
	}

	return testutil.GenerateFilterPattern(versions), nil
}

// GoldieFilesExist returns true if the required goldie files exist.
func (fixture *goTestFixture) GoldieFilesExist() bool {
	return testutil.GoldieFileExists(fixture.testdataPath, "go_list_all.golden") &&
		testutil.GoldieFileExists(fixture.testdataPath, "go_latest_stable.golden")
}
