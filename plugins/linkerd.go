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

// NewLinkerdPlugin creates a new linkerd plugin instance.
func NewLinkerdPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "linkerd",
		RepoOwner:  "linkerd",
		RepoName:   "linkerd2",
		BinaryName: "linkerd",

		FileNameTemplate:    "linkerd2-cli-stable-{{.Version}}-{{.Platform}}-{{.Arch}}",
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/stable-{{.Version}}/{{.FileName}}",
		VersionPrefix:       "stable-",
		HelpDescription:     "Linkerd - Ultralight service mesh for Kubernetes",
		HelpLink:            "https://github.com/linkerd/linkerd2",
		ArchiveType:         "none",

		VersionFilter: `^\d+\.\d+\.\d+`,
	})
}
