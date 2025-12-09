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

package testutil

import (
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

// newDummyPluginConfig returns a shared PluginTestConfig used by all testutil self-tests.
func newDummyPluginConfig(name string) *PluginTestConfig {
	binaryCfg := &asdf.BinaryPluginConfig{
		Name:            name,
		BinaryName:      "dummy-bin",
		RepoOwner:       "dummy-owner",
		RepoName:        "dummy-repo",
		HelpDescription: "Dummy tool for testing",
		HelpLink:        "https://example.com/dummy",
		ArchiveType:     "",
	}

	cfg := &PluginTestConfig{Config: binaryCfg, ForceMock: true}

	cfg.NewPlugin = func() asdf.Plugin {
		return asdf.NewBinaryPlugin(cfg.Config)
	}

	cfg.NewPluginWithClient = func(client *github.Client) asdf.Plugin {
		return asdf.NewBinaryPlugin(cfg.Config).WithGithubClient(client)
	}

	return cfg
}
