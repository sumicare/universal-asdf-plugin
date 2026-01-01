# Universal ASDF Plugin 

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](LICENSE)

**Translations:** [Українська](./doc/README.UA.md) • [Español](./doc/README.ES.md) • [Français](./doc/README.FR.md) • [Deutsch](./doc/README.DE.md) • [Polski](./doc/README.PL.md) • [Română](./doc/README.RO.md) • [Čeština](./doc/README.CS.md) • [Norsk](./doc/README.NO.md) • [한국어](./doc/README.KO.md) • [日本語](./doc/README.JA.md)

A unified collection of [asdf](https://asdf-vm.com) plugins written in Go, replacing traditional bash-scripted plugins with a single, tested, and maintainable binary.

## Why 

- **Security** — Bash plugins scattered across repositories are a valid attack surface
- **Reliability** — Go provides decent testing capabilities and reproducibility
- **Maintenance** — Single codebase for 60+ tools instead of maintaining separate plugins with kitchen-sink conventions

## Quick Start

```bash
# 1. Download the latest release
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# Or install via Go (requires Go 1.25+)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. Bootstrap asdf (installs asdf version manager itself), assuming $GOPATH/bin is in PATH already
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. Configure your shell (add to ~/.bashrc, ~/.zshrc, etc.)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. Restart your shell, then install all plugins
universal-asdf-plugin install-plugin
```

After setup, manage your tools with asdf as usual:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## Supported Tools

<details>
<summary> Click to expand full list (60+ tools)</summary>

| Tool | Description |
|------|-------------|
| [`argo`](plugins/asdf_plugin_argo) | Argo Workflows CLI |
| [`argo-rollouts`](plugins/asdf_plugin_argo_rollouts) | Argo Rollouts CLI |
| [`argocd`](plugins/asdf_plugin_argocd) | Argo CD CLI |
| [`asdf`](plugins/asdf_plugin_asdf) | asdf version manager (self-management) |
| [`awscli`](plugins/asdf_plugin_awscli) | AWS Command Line Interface |
| [`aws-nuke`](plugins/asdf_plugin_aws_nuke) | AWS resource cleanup |
| [`aws-sso-cli`](plugins/asdf_plugin_aws_sso_cli) | AWS SSO CLI |
| [`buf`](plugins/asdf_plugin_buf) | Protobuf tooling |
| [`checkov`](plugins/asdf_plugin_checkov) | Infrastructure as Code scanner |
| [`cmake`](plugins/asdf_plugin_cmake) | Cross-platform build system |
| [`cosign`](plugins/asdf_plugin_cosign) | Container signing |
| [`doctl`](plugins/asdf_plugin_doctl) | DigitalOcean CLI |
| [`gcloud`](plugins/asdf_plugin_gcloud) | Google Cloud SDK |
| [`ginkgo`](plugins/asdf_plugin_ginkgo) | Go testing framework |
| [`gitleaks`](plugins/asdf_plugin_gitleaks) | Detect secrets in code |
| [`gitsign`](plugins/asdf_plugin_gitsign) | Git commit signing |
| [`go`](plugins/asdf_plugin_go) | Go programming language |
| [`golangci-lint`](plugins/asdf_plugin_golangci_lint) | Go linters aggregator |
| [`goreleaser`](plugins/asdf_plugin_goreleaser) | Release automation |
| [`grype`](plugins/asdf_plugin_grype) | Vulnerability scanner |
| [`helm`](plugins/asdf_plugin_helm) | Kubernetes package manager |
| [`jq`](plugins/asdf_plugin_jq) | JSON processor |
| [`k9s`](plugins/asdf_plugin_k9s) | Kubernetes CLI UI |
| [`kind`](plugins/asdf_plugin_kind) | Kubernetes in Docker |
| [`ko`](plugins/asdf_plugin_ko) | Container image builder for Go |
| [`kubectl`](plugins/asdf_plugin_kubectl) | Kubernetes CLI |
| [`lazygit`](plugins/asdf_plugin_lazygit) | Git terminal UI |
| [`linkerd`](plugins/asdf_plugin_linkerd) | Service mesh CLI |
| [`nerdctl`](plugins/asdf_plugin_nerdctl) | containerd CLI |
| [`nodejs`](plugins/asdf_plugin_nodejs) | Node.js runtime |
| [`opentofu`](plugins/asdf_plugin_opentofu) | Terraform fork |
| [`pipx`](plugins/asdf_plugin_pipx) | Python app installer |
| [`protoc`](plugins/asdf_plugin_protoc) | Protocol Buffers compiler |
| [`protolint`](plugins/asdf_plugin_protolint) | Protocol Buffers linter |
| [`protoc-gen-go`](plugins/asdf_plugin_protoc_gen_go) | Go protobuf generator |
| [`protoc-gen-go-grpc`](plugins/asdf_plugin_protoc_gen_go_grpc) | gRPC Go protoc plugin |
| [`protoc-gen-grpc-web`](plugins/asdf_plugin_protoc_gen_grpc_web) | gRPC-Web protoc plugin |
| [`python`](plugins/asdf_plugin_python) | Python runtime |
| [`rust`](plugins/asdf_plugin_rust) | Rust toolchain |
| [`sccache`](plugins/asdf_plugin_sccache) | Shared compilation cache |
| [`shellcheck`](plugins/asdf_plugin_shellcheck) | Shell script analyzer |
| [`shfmt`](plugins/asdf_plugin_shfmt) | Shell formatter |
| [`sops`](plugins/asdf_plugin_sops) | Secrets manager |
| [`sqlc`](plugins/asdf_plugin_sqlc) | SQL compiler |
| [`syft`](plugins/asdf_plugin_syft) | SBOM generator |
| [`tekton-cli`](plugins/asdf_plugin_tekton_cli) | Tekton Pipelines CLI |
| [`telepresence`](plugins/asdf_plugin_telepresence) | Kubernetes dev tool |
| [`terraform`](plugins/asdf_plugin_terraform) | Infrastructure as Code |
| [`terragrunt`](plugins/asdf_plugin_terragrunt) | Terraform wrapper |
| [`terrascan`](plugins/asdf_plugin_terrascan) | IaC security scanner |
| [`tflint`](plugins/asdf_plugin_tflint) | Terraform linter |
| [`tfupdate`](plugins/asdf_plugin_tfupdate) | Terraform updater |
| [`traefik`](plugins/asdf_plugin_traefik) | Cloud-native proxy |
| [`trivy`](plugins/asdf_plugin_trivy) | Security scanner |
| [`upx`](plugins/asdf_plugin_upx) | Executable packer |
| [`uv`](plugins/asdf_plugin_uv) | Python package manager |
| [`velero`](plugins/asdf_plugin_velero) | Kubernetes backup |
| [`vultr-cli`](plugins/asdf_plugin_vultr_cli) | Vultr CLI |
| [`yq`](plugins/asdf_plugin_yq) | YAML processor |
| [`zig`](plugins/asdf_plugin_zig) | Zig programming language |

</details>

## Usage

```bash
# List available versions
universal-asdf-plugin list-all <tool>

# Install a specific version
universal-asdf-plugin install <tool> <version>

# Get the latest stable version
universal-asdf-plugin latest-stable <tool>

# Show help for a tool
universal-asdf-plugin help <tool>

# Update .tool-versions to latest versions
universal-asdf-plugin update-tool-versions
```

## Development

### Prerequisites

- Go 1.25+
- Docker (for dev container)

Mostly plugins share the same [BinaryPlugin](plugins/asdf/binary_plugin.go) interface, but there are custom ones as well:

 - [plugins/argo](plugins/argo.go) - Builds Argo Workflows.
 - [plugins/ginkgo](plugins/ginkgo.go ) - Ginkgo as well 
 - [plugins/go](plugins/golang.go) - Manages Go toolchains
 - [plugins/nodejs](plugins/nodejs.go) - Manages Node.js toolchains
 - [plugins/python](plugins/python.go) - Uses python-build / InstallFromSource to compile Python from source
 - [plugins/rust](plugins/rust.go) - Uses Rust’s own toolchain installer
 - [plugins/awscli](plugins/awscli.go) - Installs AWS CLI via its embedded Python/pip distribution
 - [plugins/gcloud](plugins/gcloud.go) - Uses Google’s Cloud SDK installer layout
 - [plugins/pipx](plugins/pipx.go) - Installs via Python/pipx mechanisms
 - [plugins/zig](plugins/zig.go) - Installs custom binary distribution

### Getting Started

```bash
# Clone the repository
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Open in VS Code with Dev Container
code universal-asdf-plugin.code-workspace

# Build locally
./scripts/build.sh
```

### Running Tests

```bash
# Update goldenfiles
./scripts/test.sh --update

# Run all smoke tests with mocked servers
./scripts/test.sh

# Run mutation tests
./scripts/mutation-test.sh

# Linting
./scripts/lint.sh

# Spellcheck
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# inspect .code-workspace dictionary afterwards
```

## License

Copyright 2025 Sumicare

By using this project, you agree to the Sumicare OSS [Terms of Use](OSS_TERMS.md).

Licensed under the [Apache License, Version 2.0](LICENSE).
