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

// NewCheckovPlugin creates a new checkov plugin instance.
func NewCheckovPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "checkov",
		RepoOwner:  "bridgecrewio",
		RepoName:   "checkov",
		BinaryName: "checkov",

		FileNameTemplate:    "checkov_{{.Platform}}_{{.Arch}}.zip",
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/{{.Version}}/{{.FileName}}",
		HelpDescription:     "Checkov - IaC security scanner",
		HelpLink:            "https://www.checkov.io/",
		ArchiveType:         "zip",
		ArchMap: map[string]string{
			"amd64": "X86_64",
			"arm64": "arm64",
		},
	})
}
