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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var testGlobalsMu sync.Mutex //nolint:gochecknoglobals // test-only global to serialize mutation of package-level vars

var (
	testGlobalsStateMu sync.Mutex              //nolint:gochecknoglobals // test-only global
	testGlobalsHeldBy  = map[*testing.T]bool{} //nolint:gochecknoglobals // test-only global
)

func lockTestGlobals(t *testing.T) {
	t.Helper()

	testGlobalsStateMu.Lock()

	if testGlobalsHeldBy[t] {
		testGlobalsStateMu.Unlock()

		return
	}

	testGlobalsHeldBy[t] = true

	testGlobalsStateMu.Unlock()

	testGlobalsMu.Lock()
	t.Cleanup(func() {
		testGlobalsStateMu.Lock()
		delete(testGlobalsHeldBy, t)
		testGlobalsStateMu.Unlock()

		testGlobalsMu.Unlock()
	})
}

// TestHelperProcess is used to mock exec.CommandContext calls.
func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]

			break
		}

		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No command")
		os.Exit(2) //nolint:revive // we're fine
	}

	cmd, args := args[0], args[1:]

	// Handle mock asdf
	if filepath.Base(cmd) != "asdf" {
		os.Exit(0) //nolint:revive // we're fine
	}

	if len(args) < 2 || args[0] != "latest" {
		os.Exit(0) //nolint:revive // we're fine
	}

	tool := args[1]

	if os.Getenv("ASDF_MOCK_ERROR") == "1" {
		os.Exit(1) //nolint:revive // we're fine
	}

	envKey := "ASDF_MOCK_VERSION_" + strings.ToUpper(tool)
	if v := os.Getenv(envKey); v != "" {
		fmt.Fprint(os.Stdout, v)
	}

	os.Exit(0) //nolint:revive // we're fine
}

// mockExec mocks exec.CommandContext to run TestHelperProcess.
func mockExec(t *testing.T, lookPath func(string) (string, error)) {
	t.Helper()
	lockTestGlobals(t)

	origLookPath := execLookPath
	origCommandContext := execCommandContext

	t.Cleanup(func() {
		execLookPath = origLookPath
		execCommandContext = origCommandContext
	})

	if lookPath != nil {
		execLookPath = lookPath
	} else {
		execLookPath = func(file string) (string, error) {
			return "/bin/" + file, nil
		}
	}

	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}

		cs = append(cs, args...)

		cmd := exec.CommandContext(ctx, os.Args[0], cs...) //nolint:gosec // it's a testing helper

		cmd.Env = append(os.Environ(), "GO_TEST_HELPER_PROCESS=1")

		return cmd
	}
}

// mockOS mocks os.Getwd and os.UserHomeDir.
func mockOS(t *testing.T, wd, home string) {
	t.Helper()
	lockTestGlobals(t)

	origGetwd := osGetwd
	origUserHomeDir := osUserHomeDir

	t.Cleanup(func() {
		osGetwd = origGetwd
		osUserHomeDir = origUserHomeDir
	})

	if wd != "" {
		osGetwd = func() (string, error) {
			return wd, nil
		}
	}

	if home != "" {
		osUserHomeDir = func() (string, error) {
			return home, nil
		}
	}
}
