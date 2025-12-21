#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

# Configuration
ASDF_DATA_DIR="${ASDF_DATA_DIR:-$HOME/.asdf}"
ASDF_BIN="${ASDF_DATA_DIR}/shims/asdf"
TOOL_VERSIONS=".tool-versions"
TOOL_SUMS=".tool-sums"
PROJECT_DIR="$(pwd)"
BINARY="${PROJECT_DIR}/build/universal-asdf-plugin"
CLEAN_INSTALL="${CLEAN_INSTALL:-false}"

export ASDF_DATA_DIR

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN_INSTALL=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--clean]"
            echo ""
            echo "Options:"
            echo "  --clean    Remove ~/.asdf completely before testing (fresh install)"
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

# Verify .tool-versions exists
[[ ! -f "${TOOL_VERSIONS}" ]] && { log_error "${TOOL_VERSIONS} not found"; exit 1; }

# Remove .tool-sums if exists
[[ -f "${TOOL_SUMS}" ]] && rm "${TOOL_SUMS}" && { log_info "Removed ${TOOL_SUMS}"; }

# Build binary
log_info "Building universal-asdf-plugin..."
./scripts/build.sh
log_info "Binary built at ${BINARY}"

# Add shims to PATH after build so plugins can find dependencies (e.g. asdf, npm, go)
export PATH="${ASDF_DATA_DIR}/shims:$PATH"

# Setup asdf directory
if [[ "${CLEAN_INSTALL}" == "true" ]]; then
    log_info "Clean install: Removing ${ASDF_DATA_DIR} (preserving downloads)..."
    rm -rf "${ASDF_DATA_DIR}/plugins" "${ASDF_DATA_DIR}/installs" "${ASDF_DATA_DIR}/shims"
fi

mkdir -p "${ASDF_DATA_DIR}/plugins" "${ASDF_DATA_DIR}/installs" "${ASDF_DATA_DIR}/shims" "${ASDF_DATA_DIR}/downloads"

# Ensure asdf itself is installed and shimmed
if [[ ! -x "${ASDF_BIN}" ]]; then
    log_info "Bootstrapping asdf via universal-asdf-plugin..."

    ASDF_INSTALL_VERSION=$(grep '^asdf ' "${TOOL_VERSIONS}" | awk '{print $2}')
    if [[ -z "${ASDF_INSTALL_VERSION}" ]]; then
        log_error "asdf version not found in ${TOOL_VERSIONS}"
        exit 1
    fi

    export ASDF_INSTALL_VERSION
    export ASDF_DOWNLOAD_PATH="${ASDF_DATA_DIR}/downloads/asdf/${ASDF_INSTALL_VERSION}"
    export ASDF_INSTALL_PATH="${ASDF_DATA_DIR}/installs/asdf/${ASDF_INSTALL_VERSION}"

    mkdir -p "${ASDF_DOWNLOAD_PATH}" "${ASDF_INSTALL_PATH}" "${ASDF_DATA_DIR}/shims"

    "${BINARY}" install-plugin asdf
    "${BINARY}" download asdf
    "${BINARY}" install asdf

    ln -sf "${ASDF_INSTALL_PATH}/bin/asdf" "${ASDF_BIN}"
fi

# Install plugins
NEED_REINSTALL=false

# Check if any installed plugin points to the wrong binary
if [[ -d "${ASDF_DATA_DIR}/plugins" ]]; then
    for plugin_bin in "${ASDF_DATA_DIR}/plugins"/*/bin/list-all; do
        [[ ! -f "$plugin_bin" ]] && continue
        
        EXISTING_PATH=$(grep -o 'exec "[^"]*"' "$plugin_bin" 2>/dev/null | head -1 | sed 's/exec "//;s/"//' || true)
        if [[ "${EXISTING_PATH}" != "${BINARY}" ]]; then
            log_warn "Plugin $(basename "$(dirname "$(dirname "$plugin_bin")")") has incorrect binary path: ${EXISTING_PATH}"
            NEED_REINSTALL=true
            break
        fi
    done
fi

if [[ "${CLEAN_INSTALL}" == "true" ]] || [[ "${NEED_REINSTALL}" == "true" ]] || [[ ! -d "${ASDF_DATA_DIR}/plugins/golang" ]]; then
    log_info "Installing universal-asdf-plugin as asdf plugins..."
    "${BINARY}" install-plugin
else
    log_info "Plugins already installed with correct binary path, skipping..."
fi

# Get list of installed plugins
log_info "Installed plugins: $(find "${ASDF_DATA_DIR}/plugins" -mindepth 1 -maxdepth 1 -type d -printf '%P ' 2>/dev/null)"

# Install tools from .tool-versions
log_info "Installing tools from ${TOOL_VERSIONS}..."

declare -i FAILED_COUNT=0 SKIPPED_COUNT=0 INSTALLED_COUNT=0 ALREADY_COUNT=0

while IFS=' ' read -r tool version rest; do
    # Skip comments and empty lines
    [[ -z "$tool" || "$tool" == \#* ]] && continue
    
    # Skip if plugin not available
    if [[ ! -d "${ASDF_DATA_DIR}/plugins/${tool}" ]]; then
        log_warn "Skipping ${tool}: plugin not available"
        ((SKIPPED_COUNT++)) || true
        continue
    fi

    # Check if already installed
    if [[ "${CLEAN_INSTALL}" != "true" ]]; then
        check_version="$version"
        if [[ "$version" == "latest" ]]; then
            check_version=$(find "${ASDF_DATA_DIR}/installs/${tool}" -mindepth 1 -maxdepth 1 -type d -printf '%P\n' 2>/dev/null | head -1 || true)
        fi
        if [[ -n "$check_version" && -d "${ASDF_DATA_DIR}/installs/${tool}/${check_version}" ]]; then
            log_info "Already installed: ${tool} ${check_version}"
            ((ALREADY_COUNT++)) || true
            continue
        fi
    fi

    # Install the tool
    log_info "Installing ${tool} ${version}..."
    if "${ASDF_BIN}" install "${tool}" "${version}"; then
        ((INSTALLED_COUNT++)) || true
    else
        log_error "Failed to install ${tool} ${version}"
        ((FAILED_COUNT++)) || true
    fi
done < "${TOOL_VERSIONS}"

# Report results
if [[ $ALREADY_COUNT -gt 0 ]]; then
    log_info "Already installed (skipped): $ALREADY_COUNT tools"
fi

if [[ $INSTALLED_COUNT -gt 0 ]]; then
    log_info "Newly installed: $INSTALLED_COUNT tools"
fi

if [[ $SKIPPED_COUNT -gt 0 ]]; then
    log_warn "Skipped (not supported): $SKIPPED_COUNT tools"
fi

if [[ $FAILED_COUNT -gt 0 ]]; then
    log_error "Failed: $FAILED_COUNT tools"
    exit 1
fi

# Reshim
log_info "Reshimming all tools..."
"${BINARY}" reshim

# Verify installations and check versions
log_info "Verifying installations and checking tool versions..."
VERIFICATION_FAILED=0
VERIFIED_COUNT=0

# Map tool names to their primary binary and version command
get_version_info() {
    local tool="$1"
    case "${tool}" in
        argocd)      echo "argocd:version --client" ;;
        argo)        echo "argo:version" ;;
        argo-rollouts) echo "kubectl-argo-rollouts:version" ;;
        asdf)        echo "asdf:--version" ;;
        awscli)      echo "aws:--version" ;;
        aws-sso-cli) echo "aws-sso:version" ;;
        cosign)      echo "cosign:version" ;;
        doctl)       echo "doctl:version" ;;
        gcloud)      echo "gcloud:--version" ;;
        ginkgo)      echo "ginkgo:version" ;;
        github-cli)  echo "gh:--version" ;;
        golang)      echo "go:version" ;;
        golangci-lint) echo "golangci-lint:--version" ;;
        goreleaser)  echo "goreleaser:--version" ;;
        grype)       echo "grype:version" ;;
        helm)        echo "helm:version --short" ;;
        jq)          echo "jq:--version" ;;
        k9s)         echo "k9s:version --short" ;;
        kind)        echo "kind:version" ;;
        ko)          echo "ko:version" ;;
        kubectl)     echo "kubectl:version --client -o yaml" ;;
        linkerd)     echo "linkerd:version --client --short" ;;
        nodejs)      echo "node:--version" ;;
        opentofu)    echo "tofu:version" ;;
        pipx)        echo "pipx:--version" ;;
        python)      echo "python3:--version" ;;
        rust)        echo "rustc:--version" ;;
        shellcheck)  echo "shellcheck:--version" ;;
        shfmt)       echo "shfmt:--version" ;;
        sops)        echo "sops:--version" ;;
        sqlc)        echo "sqlc:version" ;;
        syft)        echo "syft:version" ;;
        tekton-cli)  echo "tkn:version" ;;
        telepresence) echo "telepresence:version" ;;
        terraform)   echo "terraform:version" ;;
        terrascan)   echo "terrascan:version" ;;
        trivy)       echo "trivy:--version" ;;
        velero)      echo "velero:version --client-only" ;;
        vultr-cli)   echo "vultr-cli:version" ;;
        yq)          echo "yq:--version" ;;
        *)           echo "${tool}:--version" ;;
    esac
}

while IFS=' ' read -r tool version rest; do
    [[ -z "$tool" || "$tool" == \#* ]] && continue
    [[ ! -d "${ASDF_DATA_DIR}/plugins/${tool}" ]] && continue
    
    # Resolve latest version
    if [[ "$version" == "latest" ]]; then
        version=$(find "${ASDF_DATA_DIR}/installs/${tool}" -mindepth 1 -maxdepth 1 -type d -printf '%P\n' 2>/dev/null | sort -V | tail -1 || true)
        [[ -z "$version" ]] && continue
    fi
    
    INSTALL_DIR="${ASDF_DATA_DIR}/installs/${tool}/${version}"
    [[ ! -d "${INSTALL_DIR}" ]] && {
        log_error "Missing installation: ${tool} ${version}"
        ((VERIFICATION_FAILED++)) || true
        continue
    }
    
    # Get binary name and version command
    version_info=$(get_version_info "${tool}")
    bin_name="${version_info%%:*}"
    version_cmd="${version_info#*:}"
    
    # Check if shim exists and is executable
    shim_path="${ASDF_DATA_DIR}/shims/${bin_name}"
    if [[ ! -x "${shim_path}" ]]; then
        # Try to find any shim for this tool
        shim_path=$(find "${ASDF_DATA_DIR}/shims" -type f -name "*" -exec grep -l "ASDF_PLUGIN_NAME=\"${tool}\"" {} \; 2>/dev/null | head -1 || true)
        if [[ -z "${shim_path}" ]]; then
            log_error "No shim found for ${tool} ${version}"
            ((VERIFICATION_FAILED++)) || true
            continue
        fi
        bin_name=$(basename "${shim_path}")
    fi
    
    # Execute binary and get version
    VERSION_OUTPUT=""
    if [[ "${version_cmd}" == "--version" ]]; then
        VERSION_OUTPUT=$("${bin_name}" --version 2>&1 | head -1) || true
    elif [[ "${version_cmd}" == "version" ]]; then
        VERSION_OUTPUT=$("${bin_name}" version 2>&1 | head -1) || true
    else
        # Custom version command
        VERSION_OUTPUT=$(eval "${bin_name} ${version_cmd}" 2>&1 | head -1) || true
    fi
    
    if [[ -n "${VERSION_OUTPUT}" ]]; then
        VERSION_SHORT=$(echo "${VERSION_OUTPUT}" | cut -c 1-60)
        log_info "✓ ${tool} ${version}: ${VERSION_SHORT}"
        ((VERIFIED_COUNT++)) || true
    else
        log_warn "✓ ${tool} ${version}: installed (version check failed)"
        ((VERIFIED_COUNT++)) || true
    fi
done < "${TOOL_VERSIONS}"

log_info "Verified ${VERIFIED_COUNT} tools"

# Generate checksums
log_info "Generating checksums for installed tools..."
"${BINARY}" generate-tool-sums || log_warn "Failed to generate checksums (command may not exist)"

# Summary
echo ""
echo "========================================"
echo "           SELFTEST SUMMARY"
echo "========================================"
echo ""

if [[ $VERIFICATION_FAILED -gt 0 ]]; then
    log_error "Verification failed: $VERIFICATION_FAILED tools"
    exit 1
fi

log_info "Done - all tests passed!"
