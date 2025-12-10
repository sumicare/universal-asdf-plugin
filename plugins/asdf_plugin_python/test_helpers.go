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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/mock"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// pythonTestFixture provides plugin instances for testing.
type pythonTestFixture struct {
	plugin       *Plugin
	server       *mock.Server
	pyenvDir     string
	testdataPath string
	versions     []string
}

// newPythonTestFixture constructs a test fixture for the python plugin in the given test mode.
// When forceMock is true, creates a mock fixture even in ONLINE mode (for mock-specific tests).
func newPythonTestFixture() *pythonTestFixture {
	return newPythonTestFixtureWithMode(false)
}

// newPythonTestFixtureWithMode constructs a test fixture with explicit mode control.
func newPythonTestFixtureWithMode(forceMock bool) *pythonTestFixture {
	fixture := &pythonTestFixture{}

	_, file, _, _ := runtime.Caller(0)

	fixture.testdataPath = filepath.Join(filepath.Dir(file), "testdata")

	useMock := forceMock || !asdf.IsOnline()

	if useMock {
		fixture.server = mock.NewServer("python", "cpython")

		var err error

		fixture.pyenvDir, err = os.MkdirTemp("", "pyenv-mock-*")
		Expect(err).NotTo(HaveOccurred())

		githubClient := github.NewClientWithHTTP(nil, "https://mock-github")

		fixture.plugin = NewWithURLs(fixture.server.URL()+"/ftp/python/", githubClient)
		fixture.plugin.pyenvDir = fixture.pyenvDir
		fixture.setupMockPythonBuild()
	} else {
		fixture.plugin = New()
	}

	return fixture
}

// Close closes the mock server and cleans up temporary directories.
func (fixture *pythonTestFixture) Close() {
	if fixture.server != nil {
		fixture.server.Close()
	}

	if fixture.pyenvDir != "" {
		os.RemoveAll(fixture.pyenvDir)
	}
}

// SetupVersion sets up a specific version for testing by registering it in the mock.
func (fixture *pythonTestFixture) SetupVersion(version string) {
	fixture.versions = append(fixture.versions, version)

	fixture.server.RegisterHTML("/ftp/python/", fixture.buildVersionsHTML())

	path := fmt.Sprintf("/ftp/python/%s/Python-%s.tgz", version, version)
	fixture.server.RegisterFile(path, []byte("mock python source"))
}

// buildVersionsHTML builds a simple HTML index listing the registered Python versions for the mock FTP server.
func (fixture *pythonTestFixture) buildVersionsHTML() string {
	var sb strings.Builder
	sb.WriteString("<html><body>\n")

	for _, v := range fixture.versions {
		sb.WriteString(fmt.Sprintf(`<a href="%s/">%s/</a>`+"\n", v, v))
	}

	sb.WriteString("</body></html>")

	return sb.String()
}

// setupMockPythonBuild creates a mock python-build script in the pyenv directory used by tests.
func (fixture *pythonTestFixture) setupMockPythonBuild() {
	binDir := filepath.Join(fixture.pyenvDir, "plugins", "python-build", "bin")
	err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission)
	Expect(err).NotTo(HaveOccurred())

	script := `#!/bin/sh
# Usage:
#   python-build --definitions
#   python-build <definition> <prefix>
#   python-build --patch <definition> <prefix>

if [ "$1" = "--definitions" ]; then
	# Return a small, fixed list of versions for testing ListAll.
	# Include some non-stable or non-CPython-style definitions to verify filtering.
	echo "3.10.0"
	echo "3.11.0"
	echo "3.12.0"
	echo "3.14.1"
	echo "3.14.1t"
	echo "3.14-dev"
	echo "pypy3.10-7.3.15"
	exit 0
fi

# Handle --patch flag
if [ "$1" = "--patch" ]; then
	definition=$2
	prefix=$3
	# Read patch from stdin (ignored in mock)
	cat > /dev/null
else
	definition=$1
	prefix=$2
fi

# Handle -p flag (patch from file)
if [ "$3" = "-p" ]; then
	# Read patch from stdin (ignored in mock)
	cat > /dev/null
fi

if [ -z "$prefix" ]; then
	echo "usage: python-build <definition> <prefix>"
	exit 1
fi

# Simulate installation
echo "Installing Python $definition to $prefix..."
mkdir -p "$prefix/bin"
touch "$prefix/bin/python"
chmod +x "$prefix/bin/python"

# Create mock pip
touch "$prefix/bin/pip"
chmod +x "$prefix/bin/pip"
cat > "$prefix/bin/pip" <<EOF
#!/bin/sh
echo "pip installed packages"
EOF

echo "Python $definition installed successfully"
`

	err = os.WriteFile(filepath.Join(binDir, "python-build"), []byte(script), asdf.CommonDirectoryPermission)
	Expect(err).NotTo(HaveOccurred())
}

// SetupTagsFromGoldie reads versions from goldie test data and registers them.
// This ensures tests use the same versions as the goldie snapshots.
func (fixture *pythonTestFixture) SetupTagsFromGoldie() error {
	versions, err := testutil.ReadGoldieVersions(fixture.testdataPath, "python_list_all.golden")
	if err != nil {
		return err
	}

	for _, v := range versions {
		fixture.SetupVersion(v)
	}

	return nil
}

// GoldieVersions returns the versions from the goldie test data.
func (fixture *pythonTestFixture) GoldieVersions() ([]string, error) {
	return testutil.ReadGoldieVersions(fixture.testdataPath, "python_list_all.golden")
}

// GoldieLatest returns the latest version from the goldie test data.
func (fixture *pythonTestFixture) GoldieLatest() (string, error) {
	return testutil.ReadGoldieLatest(fixture.testdataPath, "python_latest_stable.golden")
}

// GoldieFilterPattern returns a filter pattern based on goldie versions.
func (fixture *pythonTestFixture) GoldieFilterPattern() (string, error) {
	versions, err := fixture.GoldieVersions()
	if err != nil {
		return "", err
	}

	return testutil.GenerateFilterPattern(versions), nil
}

// GoldieFilesExist returns true if the required goldie files exist.
func (fixture *pythonTestFixture) GoldieFilesExist() bool {
	return testutil.GoldieFileExists(fixture.testdataPath, "python_list_all.golden") &&
		testutil.GoldieFileExists(fixture.testdataPath, "python_latest_stable.golden")
}
