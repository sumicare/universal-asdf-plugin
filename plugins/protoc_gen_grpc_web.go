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

// NewProtocGenGrpcWebPlugin creates a new protoc-gen-grpc-web plugin instance.
func NewProtocGenGrpcWebPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "protoc-gen-grpc-web",
		RepoOwner:  "grpc",
		RepoName:   "grpc-web",
		BinaryName: "protoc-gen-grpc-web",

		FileNameTemplate:    "protoc-gen-grpc-web-{{.Version}}-{{.Platform}}-{{.Arch}}",
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/{{.Version}}/{{.FileName}}",
		VersionPrefix:       "",
		HelpDescription:     "protoc-gen-grpc-web - gRPC-Web protoc plugin",
		HelpLink:            "https://github.com/grpc/grpc-web",
		ArchiveType:         "none",
		VersionFilter:       `^\d+\.\d+\.\d+$`,
		ArchMap: map[string]string{
			"amd64": "x86_64",
			"arm64": "aarch64",
		},
	})
}
