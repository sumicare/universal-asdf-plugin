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

package asdf_plugin_ko

import (
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// config defines the plugin configuration.
//
//nolint:gochecknoglobals // plugin configuration is a static singleton shared by all instances
var config = asdf.BinaryPluginConfig{
	Name:       "ko",
	RepoOwner:  "ko-build",
	RepoName:   "ko",
	BinaryName: "ko",

	FileNameTemplate: "ko_{{.Version}}_{{.Platform}}_{{.Arch}}.tar.gz",
	OsMap: map[string]string{
		"linux":  "Linux",
		"darwin": "Darwin",
	},
	ArchMap: map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	},
	HelpDescription: "ko - Build and deploy Go applications on Kubernetes",
	HelpLink:        "https://github.com/ko-build/ko",
	ArchiveType:     "tar.gz",
}

// New creates a new ko plugin instance.
func New() asdf.Plugin {
	return asdf.NewBinaryPlugin(&config)
}

// NewWithClient creates a new ko plugin with a custom GitHub client.
func NewWithClient(client *github.Client) asdf.Plugin {
	return asdf.NewBinaryPlugin(&config).WithGithubClient(client)
}
