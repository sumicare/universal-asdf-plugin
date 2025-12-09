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

// Package testutil provides universal testing utilities for asdf binary plugins.
//
// The package provides BinaryPluginTestFixture which reduces test boilerplate
// by handling:
//   - Mock server setup for offline testing
//   - Goldie file management for snapshot testing
//   - Version management from golden files
//   - Archive type handling (raw binary, gz, tar.gz, tar.xz, zip)
//
// # Usage
//
// Define a test configuration and create a fixture factory:
//
//	var testConfig = &testutil.BinaryPluginTestConfig{
//	    PluginName:       "myplugin",
//	    RepoOwner:        "owner",
//	    RepoName:         "repo",
//	    BinaryName:       "myplugin",
//	    FileNameTemplate: "myplugin-{{.Platform}}-{{.Arch}}",
//	    ArchiveType:      "", // or "gz", "tar.gz", "tar.xz", "zip"
//	    NewPlugin:        New,
//	    NewPluginWithClient: func(client *github.Client) asdf.Plugin {
//	        return NewWithClient(client)
//	    },
//	}
//
//	func newMyPluginTestFixture() *testutil.BinaryPluginTestFixture {
//	    return testutil.NewBinaryPluginTestFixture(testConfig, 2)
//	}
//
// Then use the fixture in tests:
//
//	var fixture *testutil.BinaryPluginTestFixture
//
//	BeforeEach(func() {
//	    fixture = newMyPluginTestFixture()
//	})
//
//	AfterEach(func() {
//	    fixture.Close()
//	})
//
//	It("lists versions", func() {
//	    if !asdf.IsOnline() {
//	        if !fixture.GoldieFilesExist() {
//	            Skip("goldie files not found - run with ONLINE=1 to create")
//	        }
//	        Expect(fixture.SetupTagsFromGoldie()).To(Succeed())
//	    }
//
//	    versions, err := fixture.Plugin.ListAll(context.Background())
//	    Expect(err).NotTo(HaveOccurred())
//	    Expect(versions).NotTo(BeEmpty())
//	})
package testutil
