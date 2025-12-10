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

// NewK9sPlugin creates a new K9s plugin instance.
func NewK9sPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "k9s",
		RepoOwner:  "derailed",
		RepoName:   "k9s",
		BinaryName: "k9s",

		FileNameTemplate: "k9s_{{.Platform}}_{{.Arch}}.tar.gz",
		OsMap: map[string]string{
			"linux":  "Linux",
			"darwin": "Darwin",
		},
		HelpDescription: "K9s - Kubernetes CLI To Manage Your Clusters In Style",
		HelpLink:        "https://github.com/derailed/k9s",
		ArchiveType:     "tar.gz",
	})
}
