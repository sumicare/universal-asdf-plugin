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

package asdf_plugin_uv

import (
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// config defines the plugin configuration.
//
//nolint:gochecknoglobals // plugin configuration is a static singleton shared by all instances
var config = asdf.BinaryPluginConfig{
	Name:       "uv",
	RepoOwner:  "astral-sh",
	RepoName:   "uv",
	BinaryName: "uv",

	FileNameTemplate: "uv-{{.Arch}}-{{.Platform}}.tar.gz",

	DownloadURLTemplate: "https://github.com/{{.RepoOwner}}/{{.RepoName}}/releases/download/{{.Version}}/{{.FileName}}",
	ArchMap: map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	},
	OsMap: map[string]string{
		"linux":  "unknown-linux-gnu",
		"darwin": "apple-darwin",
	},
	HelpDescription: "uv - An extremely fast Python package and project manager",
	HelpLink:        "https://github.com/astral-sh/uv",
	ArchiveType:     "tar.gz",
}

// New creates a new uv plugin instance.
func New() asdf.Plugin {
	return asdf.NewBinaryPlugin(&config)
}

// NewWithClient creates a new uv plugin with a custom GitHub client.
func NewWithClient(client *github.Client) asdf.Plugin {
	return asdf.NewBinaryPlugin(&config).WithGithubClient(client)
}
