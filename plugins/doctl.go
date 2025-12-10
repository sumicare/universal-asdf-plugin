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

// NewDoctlPlugin creates a new doctl plugin instance.
func NewDoctlPlugin() asdf.Plugin {
	return asdf.NewBinaryPlugin(&asdf.BinaryPluginConfig{
		Name:       "doctl",
		RepoOwner:  "digitalocean",
		RepoName:   "doctl",
		BinaryName: "doctl",

		FileNameTemplate: "doctl-{{.Version}}-{{.Platform}}-{{.Arch}}.tar.gz",
		HelpDescription:  "doctl - DigitalOcean command-line interface",
		HelpLink:         "https://github.com/digitalocean/doctl",
		ArchiveType:      "tar.gz",
	})
}
