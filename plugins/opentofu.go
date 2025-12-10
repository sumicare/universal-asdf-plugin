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

// NewOpentofuPlugin creates a new opentofu plugin instance.
func NewOpentofuPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "opentofu",
		RepoOwner:  "opentofu",
		RepoName:   "opentofu",
		BinaryName: "tofu",

		FileNameTemplate: "tofu_{{.Version}}_{{.Platform}}_{{.Arch}}.tar.gz",
		HelpDescription:  "OpenTofu - The open source infrastructure as code tool",
		HelpLink:         "https://github.com/opentofu/opentofu",
		ArchiveType:      "tar.gz",
	})
}
