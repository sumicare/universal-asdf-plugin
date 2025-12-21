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
	"io"
	"testing"
)

func LockTestGlobalsForTests(t *testing.T) {
	t.Helper()
	lockTestGlobals(t)
}

func MockExecForTests(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	mockExec(t, fn)
}

func MockOSForTests(t *testing.T, wd, home string) {
	t.Helper()
	mockOS(t, wd, home)
}

func IsPathWithinDirForTests(path, dir string) bool {
	return isPathWithinDir(path, dir)
}

func NewLimitedArchiveWriterForTests(w io.Writer, total *int64, maxTotal, maxFile int64) io.Writer {
	return &limitedArchiveWriter{
		w:        w,
		total:    total,
		maxTotal: maxTotal,
		maxFile:  maxFile,
	}
}

func EnsureToolVersionLineForTests(toolVersionsPath, tool, version string) error {
	return ensureToolVersionLine(toolVersionsPath, tool, version)
}

func InstallDependenciesForTests(ctx context.Context, tools ...string) error {
	return installDependencies(ctx, tools...)
}

func ResolveVersionFromProjectToolVersionsForTests(tool string) string {
	return resolveVersionFromProjectToolVersions(tool)
}

func SetOSGetwdForTests(fn func() (string, error)) func() {
	orig := osGetwd
	osGetwd = fn

	return func() { osGetwd = orig }
}

func SetOSUserHomeDirForTests(fn func() (string, error)) func() {
	orig := osUserHomeDir
	osUserHomeDir = fn

	return func() { osUserHomeDir = orig }
}

func RenderSourceBuildTemplateForTests(
	tmpl string,
	cfg *SourceBuildPluginConfig,
	version string,
) string {
	return renderSourceBuildTemplate(tmpl, cfg, version)
}

func ErrSourceBuildNoBuildStepForTests() error {
	return errSourceBuildNoBuildStep
}

func ErrSourceBuildNoVersionsFoundForTests() error {
	return errSourceBuildNoVersionsFound
}

func ErrSourceBuildNoVersionsMatchingForTests() error {
	return errSourceBuildNoVersionsMatching
}
