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

// NewTrivyPlugin creates a new Trivy plugin instance.
func NewTrivyPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "trivy",
		RepoOwner:  "aquasecurity",
		RepoName:   "trivy",
		BinaryName: "trivy",

		FileNameTemplate: "trivy_{{.Version}}_{{.Platform}}-{{.Arch}}.tar.gz",
		OsMap: map[string]string{
			"linux":  "Linux",
			"darwin": "macOS",
		},
		ArchMap: map[string]string{
			"amd64": "64bit",
			"arm64": "ARM64",
		},
		HelpDescription: "Trivy - A vulnerability scanner",
		HelpLink:        "https://github.com/aquasecurity/trivy",
		ArchiveType:     "tar.gz",
	})
}
