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

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_argo"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_argo_rollouts"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_argocd"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_aws_nuke"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_aws_sso_cli"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_awscli"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_buf"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_checkov"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_cmake"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_cosign"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_doctl"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_gcloud"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_ginkgo"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_github_cli"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_gitleaks"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_gitsign"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_go"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_golangci_lint"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_goreleaser"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_grype"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_helm"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_jq"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_k9s"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_kind"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_ko"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_kubectl"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_lazygit"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_linkerd"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_nerdctl"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_nodejs"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_opentofu"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_pipx"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_protoc"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_protoc_gen_go"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_protoc_gen_go_grpc"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_protoc_gen_grpc_web"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_protolint"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_python"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_rust"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_sccache"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_shellcheck"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_shfmt"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_sops"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_sqlc"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_syft"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_tekton_cli"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_telepresence"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_terraform"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_terragrunt"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_terrascan"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_tflint"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_tfupdate"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_traefik"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_trivy"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_upx"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_uv"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_velero"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_vultr_cli"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_yq"
	"github.com/sumicare/universal-asdf-plugin/plugins/asdf_plugin_zig"
)

// errUnknownPlugin is returned when the requested plugin is not registered.
// It is used to wrap unknown plugin names in a consistent error value.
// Callers can match against this sentinel to distinguish real registry
// misses from other failure modes.
var errUnknownPlugin = errors.New("unknown plugin")

// GetPlugin returns the implementation for the given plugin name.
// It returns an error if the plugin is unknown so callers can surface
// a helpful message to users of the CLI.
//
// The mapping is intentionally centralized here so that new plugins only
// need to be registered in a single place.
//
//nolint:ireturn // factory function intentionally returns the asdf.Plugin interface
func GetPlugin(name string) (asdf.Plugin, error) {
	switch strings.ToLower(name) {
	case "argo":
		return asdf_plugin_argo.New(), nil
	case "argocd":
		return asdf_plugin_argocd.New(), nil
	case "argo-rollouts":
		return asdf_plugin_argo_rollouts.New(), nil
	case "checkov":
		return asdf_plugin_checkov.New(), nil
	case "cmake":
		return asdf_plugin_cmake.New(), nil
	case "cosign":
		return asdf_plugin_cosign.New(), nil
	case "doctl":
		return asdf_plugin_doctl.New(), nil
	case "jq":
		return asdf_plugin_jq.New(), nil
	case "k9s":
		return asdf_plugin_k9s.New(), nil
	case "kind":
		return asdf_plugin_kind.New(), nil
	case "ko":
		return asdf_plugin_ko.New(), nil
	case "kubectl":
		return asdf_plugin_kubectl.New(), nil
	case "lazygit":
		return asdf_plugin_lazygit.New(), nil
	case "linkerd":
		return asdf_plugin_linkerd.New(), nil
	case "nerdctl":
		return asdf_plugin_nerdctl.New(), nil
	case "ginkgo":
		return asdf_plugin_ginkgo.New(), nil
	case "github-cli", "gh":
		return asdf_plugin_github_cli.New(), nil
	case "gitsign":
		return asdf_plugin_gitsign.New(), nil
	case "gitleaks":
		return asdf_plugin_gitleaks.New(), nil
	case "goreleaser":
		return asdf_plugin_goreleaser.New(), nil
	case "golang", "go":
		return asdf_plugin_go.New(), nil
	case "golangci-lint":
		return asdf_plugin_golangci_lint.New(), nil
	case "grype":
		return asdf_plugin_grype.New(), nil
	case "gcloud":
		return asdf_plugin_gcloud.New(), nil
	case "aws-nuke":
		return asdf_plugin_aws_nuke.New(), nil
	case "aws-sso-cli":
		return asdf_plugin_aws_sso_cli.New(), nil
	case "awscli":
		return asdf_plugin_awscli.New(), nil
	case "buf":
		return asdf_plugin_buf.New(), nil
	case "helm":
		return asdf_plugin_helm.New(), nil
	case "python":
		return asdf_plugin_python.New(), nil
	case "pipx":
		return asdf_plugin_pipx.New(), nil
	case "rust":
		return asdf_plugin_rust.New(), nil
	case "sccache":
		return asdf_plugin_sccache.New(), nil
	case "shellcheck":
		return asdf_plugin_shellcheck.New(), nil
	case "sops":
		return asdf_plugin_sops.New(), nil
	case "shfmt":
		return asdf_plugin_shfmt.New(), nil
	case "syft":
		return asdf_plugin_syft.New(), nil
	case "terraform":
		return asdf_plugin_terraform.New(), nil
	case "terragrunt":
		return asdf_plugin_terragrunt.New(), nil
	case "terrascan":
		return asdf_plugin_terrascan.New(), nil
	case "tfupdate":
		return asdf_plugin_tfupdate.New(), nil
	case "tflint":
		return asdf_plugin_tflint.New(), nil
	case "trivy":
		return asdf_plugin_trivy.New(), nil
	case "vultr-cli":
		return asdf_plugin_vultr_cli.New(), nil
	case "nodejs", "node":
		return asdf_plugin_nodejs.New(), nil
	case "opentofu":
		return asdf_plugin_opentofu.New(), nil
	case "protoc":
		return asdf_plugin_protoc.New(), nil
	case "protoc-gen-go":
		return asdf_plugin_protoc_gen_go.New(), nil
	case "protoc-gen-go-grpc", "asdf-protoc-gen-go-grpc":
		return asdf_plugin_protoc_gen_go_grpc.New(), nil
	case "protoc-gen-grpc-web":
		return asdf_plugin_protoc_gen_grpc_web.New(), nil
	case "protolint":
		return asdf_plugin_protolint.New(), nil
	case "sqlc":
		return asdf_plugin_sqlc.New(), nil
	case "tekton-cli":
		return asdf_plugin_tekton_cli.New(), nil
	case "telepresence":
		return asdf_plugin_telepresence.New(), nil
	case "traefik":
		return asdf_plugin_traefik.New(), nil
	case "velero":
		return asdf_plugin_velero.New(), nil
	case "upx":
		return asdf_plugin_upx.New(), nil
	case "uv":
		return asdf_plugin_uv.New(), nil
	case "yq":
		return asdf_plugin_yq.New(), nil
	case "zig":
		return asdf_plugin_zig.New(), nil
	case "asdf":
		return asdf_plugin_asdf.New(), nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnknownPlugin, name)
	}
}
