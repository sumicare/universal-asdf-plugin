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

package plugins

import (
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// NewShellcheckPlugin creates a new shellcheck plugin instance.
func NewShellcheckPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "shellcheck",
		RepoOwner:  "koalaman",
		RepoName:   "shellcheck",
		BinaryName: "shellcheck",

		FileNameTemplate:    "shellcheck-v{{.Version}}.{{.Platform}}.{{.Arch}}.tar.xz",
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/v{{.Version}}/{{.FileName}}",
		HelpDescription:     "ShellCheck - A static analysis tool for shell scripts",
		HelpLink:            "https://github.com/koalaman/shellcheck",
		ArchiveType:         "tar.xz",
		ArchMap: map[string]string{
			"amd64": "x86_64",
			"arm64": "aarch64",
		},
	})
}
