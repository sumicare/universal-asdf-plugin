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

package testutil

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("dummy plugin shared ginkgo specs", func() {
	var originalOnline string

	BeforeEach(func() {
		originalOnline = os.Getenv("ONLINE")

		_ = os.Unsetenv("ONLINE")
	})

	AfterEach(func() {
		if originalOnline == "" {
			_ = os.Unsetenv("ONLINE")
		} else {
			_ = os.Setenv("ONLINE", originalOnline)
		}
	})

	cfg := newDummyPluginConfig("dummy-plugin") //nolint:ginkgolinter // it's a wrapper for declarative testutil
	cfg.TestdataPath = "testdata"

	DescribeBasicPluginBehavior(cfg)
	DescribeListAll(cfg)
	DescribeLatestStable(cfg)
	DescribeDownload(cfg)
	DescribeInstall(cfg)
	DescribeDownloadErrors(cfg)
	DescribeInstallErrors(cfg)
	DescribeMockOnlyListAll(cfg)
	DescribeMockOnlyLatestStable(cfg)
	DescribeMockOnlyDownload(cfg)
	DescribeMockOnlyInstall(cfg)
})
