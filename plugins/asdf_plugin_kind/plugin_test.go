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

package asdf_plugin_kind

import (
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf/testutil"
)

// testdataPath returns the path to this plugin's testdata directory.
func testdataPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// pluginTestConfig returns the test configuration for the kind plugin.
func pluginTestConfig() *testutil.PluginTestConfig {
	return &testutil.PluginTestConfig{
		Config:              &config,
		TestdataPath:        testdataPath(),
		NewPlugin:           New,
		NewPluginWithClient: NewWithClient,
	}
}

var _ = Describe("Kind Plugin", func() {
	cfg := pluginTestConfig() //nolint:ginkgolinter // we're programmatically bootstrapping test suite
	testutil.DescribeBasicPluginBehavior(cfg)
	testutil.DescribeListAll(cfg)
	testutil.DescribeLatestStable(cfg)
	testutil.DescribeDownload(cfg)
	testutil.DescribeInstall(cfg)
	testutil.DescribeDownloadErrors(cfg)
	testutil.DescribeInstallErrors(cfg)
})
