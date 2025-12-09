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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

var _ = Describe("GetPlugin", func() {
	It("returns registered plugins with help", func() {
		pluginNames := []string{
			"argo",
			"argocd",
			"argo-rollouts",
			"checkov",
			"cmake",
			"cosign",
			"doctl",
			"jq",
			"k9s",
			"kind",
			"ko",
			"kubectl",
			"lazygit",
			"linkerd",
			"nerdctl",
			"ginkgo",
			"github-cli",
			"gh",
			"gitsign",
			"gitleaks",
			"goreleaser",
			"golang",
			"go",
			"golangci-lint",
			"grype",
			"gcloud",
			"aws-nuke",
			"aws-sso-cli",
			"awscli",
			"buf",
			"helm",
			"python",
			"pipx",
			"rust",
			"sccache",
			"shellcheck",
			"sops",
			"shfmt",
			"syft",
			"terraform",
			"terragrunt",
			"terrascan",
			"tfupdate",
			"tflint",
			"trivy",
			"vultr-cli",
			"nodejs",
			"node",
			"opentofu",
			"protoc",
			"protoc-gen-go",
			"protoc-gen-go-grpc",
			"asdf-protoc-gen-go-grpc",
			"protoc-gen-grpc-web",
			"protolint",
			"sqlc",
			"tekton-cli",
			"telepresence",
			"traefik",
			"velero",
			"upx",
			"uv",
			"yq",
			"zig",
			"asdf",
		}

		for _, name := range pluginNames {
			plugin, err := GetPlugin(name)
			Expect(err).NotTo(HaveOccurred(), "expected plugin %s to be registered", name)
			Expect(plugin).NotTo(BeNil(), "expected plugin %s to be non-nil", name)

			help := plugin.Help()
			Expect(help.Overview).NotTo(BeEmpty(), "expected help overview for %s", name)
			Expect(help.Deps).To(BeAssignableToTypeOf(asdf.PluginHelp{}.Deps))
			Expect(help.Config).To(BeAssignableToTypeOf(asdf.PluginHelp{}.Config))
			Expect(help.Links).To(BeAssignableToTypeOf(asdf.PluginHelp{}.Links))
		}
	})

	It("returns an error for unknown plugins", func() {
		plugin, err := GetPlugin("this-plugin-does-not-exist")
		Expect(plugin).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, errUnknownPlugin)).To(BeTrue())
	})
})
