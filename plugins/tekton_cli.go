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

// NewTektonCliPlugin creates a new Tekton CLI plugin instance.
func NewTektonCliPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "tekton-cli",
		RepoOwner:  "tektoncd",
		RepoName:   "cli",
		BinaryName: "tkn",

		FileNameTemplate: "tkn_{{.Version}}_{{.Platform}}_{{.Arch}}.tar.gz",
		HelpDescription:  "Tekton CLI - CLI for interacting with Tekton",
		HelpLink:         "https://tekton.dev/",
		ArchiveType:      "tar.gz",
		VersionFilter:    `^\d+\.\d+\.\d+$`,
		OsMap: map[string]string{
			"linux":  "Linux",
			"darwin": "Darwin",
		},
		ArchMap: map[string]string{
			"amd64": "x86_64",
			"arm64": "aarch64",
		},
	})
}
