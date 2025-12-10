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

// NewProtocGenGoGrpcPlugin creates a new protoc-gen-go-grpc plugin instance.
func NewProtocGenGoGrpcPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "protoc-gen-go-grpc",
		RepoOwner:  "grpc",
		RepoName:   "grpc-go",
		BinaryName: "protoc-gen-go-grpc",

		FileNameTemplate:    "protoc-gen-go-grpc.v{{.Version}}.{{.Platform}}.{{.Arch}}.tar.gz",
		DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/cmd/protoc-gen-go-grpc/v{{.Version}}/{{.FileName}}",
		VersionPrefix:       "cmd/protoc-gen-go-grpc/v",
		HelpDescription:     "protoc-gen-go-grpc - gRPC Go protoc plugin",
		HelpLink:            "https://grpc.io/docs/languages/go/",
		ArchiveType:         "tar.gz",
		VersionFilter:       `^\d+\.\d+\.\d+$`,
		UseTags:             true,
	})
}
