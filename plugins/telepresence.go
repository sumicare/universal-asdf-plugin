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

// NewTelepresencePlugin creates a new Telepresence plugin instance.
func NewTelepresencePlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "telepresence",
		RepoOwner:  "telepresenceio",
		RepoName:   "telepresence",
		BinaryName: "telepresence",

		FileNameTemplate: "telepresence-{{.Platform}}-{{.Arch}}",
		HelpDescription:  "Telepresence - Local development against a remote Kubernetes cluster",
		HelpLink:         "https://www.telepresence.io/",
		ArchiveType:      "none",
		VersionFilter:    `^\d+\.\d+\.\d+$`,
	})
}
