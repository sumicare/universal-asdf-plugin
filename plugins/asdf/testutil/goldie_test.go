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
	"path/filepath"
	"testing"
)

// TestGoldieHelpers tests the goldie helpers.
func TestGoldieHelpers(t *testing.T) {
	testdataDir, cleanup := CreateTestDir(t)
	defer cleanup()

	versionsFile := "tool_list_all.golden"
	latestFile := "tool_latest_stable.golden"

	versionsPath := filepath.Join(testdataDir, versionsFile)
	latestPath := filepath.Join(testdataDir, latestFile)

	if GoldieFileExists(testdataDir, versionsFile) {
		t.Fatal("GoldieFileExists unexpectedly true before file creation")
	}

	if err := os.WriteFile(versionsPath, []byte("1.2.3\n1.2.4"), CommonFilePermission); err != nil {
		t.Fatalf("failed to write versions file: %v", err)
	}

	if err := os.WriteFile(latestPath, []byte("1.2.4\n"), CommonFilePermission); err != nil {
		t.Fatalf("failed to write latest file: %v", err)
	}

	if !GoldieFileExists(testdataDir, versionsFile) {
		t.Fatal("GoldieFileExists returned false after file creation")
	}

	versions, err := ReadGoldieVersions(testdataDir, versionsFile)
	if err != nil {
		t.Fatalf("ReadGoldieVersions returned error: %v", err)
	}

	if len(versions) != 2 || versions[0] != "1.2.3" || versions[1] != "1.2.4" {
		t.Fatalf("ReadGoldieVersions returned unexpected versions: %#v", versions)
	}

	latest, err := ReadGoldieLatest(testdataDir, latestFile)
	if err != nil {
		t.Fatalf("ReadGoldieLatest returned error: %v", err)
	}

	if latest != "1.2.4" {
		t.Fatalf("ReadGoldieLatest returned unexpected value: %q", latest)
	}

	if err := os.WriteFile(versionsPath, []byte("\n"), CommonFilePermission); err != nil {
		t.Fatalf("failed to overwrite versions file: %v", err)
	}

	versions, err = ReadGoldieVersions(testdataDir, versionsFile)
	if err != nil {
		t.Fatalf("ReadGoldieVersions on empty file returned error: %v", err)
	}

	if len(versions) != 0 {
		t.Fatalf("ReadGoldieVersions on empty file expected empty slice, got %#v", versions)
	}
}

// TestVersionUtilities tests the version utilities.
func TestVersionUtilities(t *testing.T) {
	if got := MaximizeVersion("3.7.1"); got != "9.9.9" {
		t.Fatalf("MaximizeVersion simple case: got %q", got)
	}

	if got := MaximizeVersion("v24.9.1"); got != "v99.9.9" {
		t.Fatalf("MaximizeVersion prefixed case: got %q", got)
	}

	if got := MaximizeVersion("stable-24"); got != "stable-99" {
		t.Fatalf("MaximizeVersion suffix case: got %q", got)
	}

	versions := []string{"1.2.3", "2.0.0"}

	tags := VersionsToTags(versions, true)
	if len(tags) != 2 || tags[0] != "v1.2.3" || tags[1] != "v2.0.0" {
		t.Fatalf("VersionsToTags with prefix returned %#v", tags)
	}

	tags = VersionsToTags([]string{"v1.0.0"}, true)
	if len(tags) != 1 || tags[0] != "v1.0.0" {
		t.Fatalf("VersionsToTags should not double-prefix versions: %#v", tags)
	}

	if pattern := GenerateFilterPattern(nil); pattern != "" {
		t.Fatalf("GenerateFilterPattern for nil slice expected empty string, got %q", pattern)
	}

	if pattern := GenerateFilterPattern([]string{"only"}); pattern != "" {
		t.Fatalf("GenerateFilterPattern single element expected empty string, got %q", pattern)
	}

	if pattern := GenerateFilterPattern([]string{"3.6.0", "3.7.0"}); pattern != "3.6" {
		t.Fatalf("GenerateFilterPattern multi element: got %q", pattern)
	}
}
