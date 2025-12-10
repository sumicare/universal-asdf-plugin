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
	"errors"
	"fmt"
	"strings"

	p "github.com/sumicare/universal-asdf-plugin/plugins"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

type (
	// PluginEntry represents a single plugin registration with its names and factory.
	PluginEntry struct {
		Factory func() asdf.Plugin
		Names   []string
	}

	// Registry holds all registered plugins and provides lookup by name.
	Registry struct {
		entries map[string]*PluginEntry
		all     []*PluginEntry
	}
)

// NewRegistry creates and initializes the plugin registry with all plugins.
//
//nolint:gocritic // factory wrappers ensure we return the asdf.Plugin interface
func NewRegistry() *Registry {
	registry := &Registry{
		entries: make(map[string]*PluginEntry),
		all:     make([]*PluginEntry, 0),
	}

	// Register all plugins
	registry.register(&PluginEntry{
		Names:   []string{"argo"},
		Factory: p.NewArgoPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"argocd"},
		Factory: p.NewArgoCDPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"argo-rollouts"},
		Factory: p.NewArgoRolloutsPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"checkov"},
		Factory: p.NewCheckovPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"cmake"},
		Factory: p.NewCmakePlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"cosign"},
		Factory: p.NewCosignPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"doctl"},
		Factory: p.NewDoctlPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"jq"},
		Factory: p.NewJqPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"k9s"},
		Factory: p.NewK9sPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"kind"},
		Factory: p.NewKindPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"ko"},
		Factory: p.NewKoPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"kubectl"},
		Factory: p.NewKubectlPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"lazygit"},
		Factory: p.NewLazygitPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"linkerd"},
		Factory: p.NewLinkerdPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"nerdctl"},
		Factory: p.NewNerdctlPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"ginkgo"},
		Factory: p.NewGinkgoPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"github-cli", "gh"},
		Factory: p.NewGithubCliPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"gitsign"},
		Factory: p.NewGitsignPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"gitleaks"},
		Factory: p.NewGitleaksPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"goreleaser"},
		Factory: p.NewGoreleaserPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"golang", "go"},
		Factory: p.NewGolangPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"golangci-lint"},
		Factory: p.NewGolangciLintPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"grype"},
		Factory: p.NewGrypePlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"gcloud"},
		Factory: p.NewGcloudPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"aws-nuke"},
		Factory: func() asdf.Plugin { return p.NewAwsNukePlugin() },
	})
	registry.register(&PluginEntry{
		Names:   []string{"aws-sso-cli"},
		Factory: p.NewAwsSsoCliPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"awscli"},
		Factory: p.NewAwscliPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"buf"},
		Factory: p.NewBufPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"helm"},
		Factory: p.NewHelmPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"python"},
		Factory: p.NewPythonPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"pipx"},
		Factory: p.NewPipxPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"rust"},
		Factory: p.NewRustPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"sccache"},
		Factory: p.NewSccachePlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"shellcheck"},
		Factory: p.NewShellcheckPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"sops"},
		Factory: p.NewSopsPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"shfmt"},
		Factory: p.NewShfmtPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"syft"},
		Factory: p.NewSyftPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"terraform"},
		Factory: p.NewTerraformPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"terragrunt"},
		Factory: p.NewTerragruntPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"terrascan"},
		Factory: p.NewTerrascanPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"tfupdate"},
		Factory: p.NewTfupdatePlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"tflint"},
		Factory: p.NewTflintPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"trivy"},
		Factory: p.NewTrivyPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"vultr-cli"},
		Factory: p.NewVultrCliPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"nodejs", "node"},
		Factory: p.NewNodejsPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"opentofu"},
		Factory: p.NewOpentofuPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"protoc"},
		Factory: p.NewProtocPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"protoc-gen-go"},
		Factory: p.NewProtocGenGoPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"protoc-gen-go-grpc", "asdf-protoc-gen-go-grpc"},
		Factory: p.NewProtocGenGoGrpcPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"protoc-gen-grpc-web"},
		Factory: p.NewProtocGenGrpcWebPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"protolint"},
		Factory: p.NewProtolintPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"sqlc"},
		Factory: p.NewSqlcPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"tekton-cli"},
		Factory: p.NewTektonCliPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"telepresence"},
		Factory: p.NewTelepresencePlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"traefik"},
		Factory: p.NewTraefikPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"velero"},
		Factory: p.NewVeleroPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"upx"},
		Factory: p.NewUpxPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"uv"},
		Factory: p.NewUvPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"yq"},
		Factory: p.NewYqPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"zig"},
		Factory: p.NewZigPlugin,
	})
	registry.register(&PluginEntry{
		Names:   []string{"asdf"},
		Factory: p.NewAsdfPlugin,
	})

	return registry
}

// register adds a plugin entry to the registry.
func (r *Registry) register(entry *PluginEntry) {
	r.all = append(r.all, entry)
	for _, name := range entry.Names {
		r.entries[strings.ToLower(name)] = entry
	}
}

// Get retrieves a plugin by name, returning nil if not found.
func (r *Registry) Get(name string) asdf.Plugin {
	entry, ok := r.entries[strings.ToLower(name)]
	if !ok {
		return nil
	}

	return entry.Factory()
}

// All returns all registered plugin entries in order.
func (r *Registry) All() []*PluginEntry {
	return r.all
}

// DefaultRegistry is the global plugin registry.
var (
	DefaultRegistry = NewRegistry() //nolint:gochecknoglobals // global registry for plugin access

	// errUnknownPlugin is returned when the requested plugin is not registered.
	errUnknownPlugin = errors.New("unknown plugin")
)

// GetPlugin returns the implementation for the given plugin name.
// It returns an error if the plugin is unknown so callers can surface
// a helpful message to users of the CLI.
//
//nolint:ireturn // factory function intentionally returns the asdf.Plugin interface
func GetPlugin(name string) (asdf.Plugin, error) {
	plugin := DefaultRegistry.Get(name)
	if plugin == nil {
		return nil, fmt.Errorf("%w: %s", errUnknownPlugin, name)
	}

	return plugin, nil
}

// GetPluginRegistry returns the global plugin registry for iteration and testing.
func GetPluginRegistry() *Registry {
	return DefaultRegistry
}
