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

// NewUpxPlugin creates a new upx plugin instance.
func NewUpxPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "upx",
		RepoOwner:  "upx",
		RepoName:   "upx",
		BinaryName: "upx",

		FileNameTemplate:    "upx-{{.Version}}-{{.Arch}}_{{.Platform}}.tar.xz",
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/v{{.Version}}/{{.FileName}}",
		HelpDescription:     "UPX - the Ultimate Packer for eXecutables",
		HelpLink:            "https://github.com/upx/upx",
		ArchiveType:         "tar.xz",
	})
}
