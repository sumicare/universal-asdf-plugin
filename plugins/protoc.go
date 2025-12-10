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

// NewProtocPlugin creates a new protoc plugin instance.
func NewProtocPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "protoc",
		RepoOwner:  "protocolbuffers",
		RepoName:   "protobuf",
		BinaryName: "protoc",

		FileNameTemplate: "protoc-{{.Version}}-{{.Platform}}-{{.Arch}}.zip",
		OsMap: map[string]string{
			"linux":  "linux",
			"darwin": "osx",
		},
		ArchMap: map[string]string{
			"amd64": "x86_64",
			"arm64": "aarch_64",
		},
		HelpDescription: "Protocol Buffers - Google's data interchange format",
		HelpLink:        "https://github.com/protocolbuffers/protobuf",
		ArchiveType:     "zip",
	})
}
