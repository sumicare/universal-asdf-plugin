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

// NewJqPlugin creates a new jq plugin instance.
func NewJqPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "jq",
		RepoOwner:  "jqlang",
		RepoName:   "jq",
		BinaryName: "jq",

		FileNameTemplate: "jq-{{.Platform}}{{.Arch}}",

		VersionPrefix:       "jq-",
		VersionFilter:       `^[0-9]+\.[0-9]+(\.[0-9]+)?$`,
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/jq-{{.Version}}/{{.FileName}}",
		ArchMap: map[string]string{
			"amd64": "64",
			"arm64": "arm64",
		},
		OsMap: map[string]string{
			"linux":  "linux",
			"darwin": "osx-",
		},
		HelpDescription: "jq - Command-line JSON processor",
		HelpLink:        "https://github.com/jqlang/jq",
		ArchiveType:     "none",
	})
}
