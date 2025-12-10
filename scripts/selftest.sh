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
PARALLEL_JOBS="${PARALLEL_JOBS:-16}"

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
            echo "Usage: $0 [--clean] [--jobs N]"
            echo ""
            echo "Options:"
            echo "  --clean    Remove ~/.asdf completely before testing (fresh install)"
            echo "  --jobs N   Number of parallel jobs (default: 16)"
            echo ""
            echo "Requires: GNU parallel (apt-get install parallel)"
            exit 0
            ;;
        --jobs)
            PARALLEL_JOBS="${2:-16}"
            shift 2
            ;;
        --jobs=*)
            PARALLEL_JOBS="${1#*=}"
            shift
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
if [[ -d "${ASDF_DATA_DIR}/plugins/golang/bin" ]]; then
    EXISTING_PATH=$(grep -o 'exec "[^"]*"' "${ASDF_DATA_DIR}/plugins/golang/bin/list-all" 2>/dev/null | head -1 | sed 's/exec "//;s/"//' || true)
    [[ "${EXISTING_PATH}" != "${BINARY}" ]] && NEED_REINSTALL=true
fi

if [[ "${CLEAN_INSTALL}" == "true" ]] || [[ "${NEED_REINSTALL}" == "true" ]] || [[ ! -d "${ASDF_DATA_DIR}/plugins/golang" ]]; then
    log_info "Installing universal-asdf-plugin as asdf plugins..."
    "${BINARY}" install-plugin
else
    log_info "Plugins already installed with correct binary path, skipping..."
fi

# Get list of installed plugins
log_info "Installed plugins: $(find "${ASDF_DATA_DIR}/plugins" -mindepth 1 -maxdepth 1 -type d -printf '%P ' 2>/dev/null)"

# Install tools
log_info "Installing tools from ${TOOL_VERSIONS} (parallel jobs: ${PARALLEL_JOBS})..."

# Check for GNU parallel
if ! command -v parallel >/dev/null 2>&1; then
    log_error "GNU parallel not found. Please install it: apt-get install parallel"
    exit 1
fi

# Function for parallel execution
install_tool() {
    local tool="$1" version="$2"

    if [[ ! -d "${ASDF_DATA_DIR}/plugins/${tool}" ]]; then
        echo "skipped:${tool}:plugin_not_available"
        return 0
    fi

    # Check if already installed
    if [[ "${CLEAN_INSTALL}" != "true" ]]; then
        local check_version="$version"
        [[ "$version" == "latest" ]] && check_version=$(find "${ASDF_DATA_DIR}/installs/${tool}" -mindepth 1 -maxdepth 1 -type d -printf '%P\n' 2>/dev/null | head -1 || true)
        [[ -n "$check_version" && -d "${ASDF_DATA_DIR}/installs/${tool}/${check_version}" ]] && {
            echo "already:${tool}:${check_version}"
            return 0
        }
    fi

    # Install
    if "${ASDF_BIN}" install "${tool}" "${version}" >&2; then
        echo "installed:${tool}:${version}"
    else
        echo "failed:${tool}:${version}"
    fi
}
export -f install_tool
export ASDF_DATA_DIR ASDF_BIN CLEAN_INSTALL

# Collect results and run in parallel
declare -i FAILED_COUNT=0 SKIPPED_COUNT=0 INSTALLED_COUNT=0 ALREADY_COUNT=0

# Filter out comments and empty lines, then run in parallel
while IFS= read -r result; do
    [[ -z "$result" ]] && continue
    status="${result%%:*}"
    rest="${result#*:}"
    tool="${rest%%:*}"
    version="${rest#*:}"

    case "$status" in
        installed)
            ((INSTALLED_COUNT++)) || true
            ;;
        already)
            ((ALREADY_COUNT++)) || true
            ;;
        skipped)
            ((SKIPPED_COUNT++)) || true
            ;;
        failed)
            ((FAILED_COUNT++)) || true
            log_error "Failed to install ${tool} ${version}"
            ;;
    esac
done < <(grep -v '^[[:space:]]*#' "${TOOL_VERSIONS}" | grep -v '^[[:space:]]*$' | \
    SHELL=/bin/bash parallel --no-notice -j "${PARALLEL_JOBS}" --colsep ' ' 'install_tool {1} {2}')

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

# Verify installations
log_info "Verifying shims and installations..."
VERIFICATION_FAILED=0

while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" =~ ^# ]] && continue
    tool=$(echo "$line" | awk '{print $1}')
    version=$(echo "$line" | awk '{print $2}')
    [[ -z "$tool" || -z "$version" ]] && continue
    
    # Skip tools without plugins
    [[ ! -d "${ASDF_DATA_DIR}/plugins/${tool}" ]] && continue
    
    # Resolve latest
    [[ "$version" == "latest" ]] && {
        actual_version=$(find "${ASDF_DATA_DIR}/installs/${tool}" -mindepth 1 -maxdepth 1 -type d -printf '%P\n' 2>/dev/null | head -1 || true)
        [[ -n "$actual_version" ]] && version="$actual_version"
    }
    
    # Check installation exists
    INSTALL_DIR="${ASDF_DATA_DIR}/installs/${tool}/${version}"
    [[ ! -d "${INSTALL_DIR}" ]] && {
        log_error "Installation directory missing: ${INSTALL_DIR}"
        ((VERIFICATION_FAILED++)) || true
        continue
    }
    
    # Check for shims
    BIN_PATHS=$(ASDF_PLUGIN_NAME="${tool}" ASDF_INSTALL_PATH="${INSTALL_DIR}" "${BINARY}" list-bin-paths 2>/dev/null || echo "bin")
    
    FOUND_SHIM=false
    for bin_path in ${BIN_PATHS}; do
        BIN_DIR="${INSTALL_DIR}/${bin_path}"
        [[ -d "${BIN_DIR}" ]] && for binary in "${BIN_DIR}"/*; do
            [[ -x "${binary}" && -f "${binary}" ]] && {
                binary_name=$(basename "${binary}")
                shim_path="${ASDF_DATA_DIR}/shims/${binary_name}"
                [[ -f "${shim_path}" ]] && FOUND_SHIM=true && break 2
            }
        done
    done
    
    if [[ "${FOUND_SHIM}" == "true" ]]; then
        log_info "Verified: ${tool} ${version} - shims OK"
    else
        log_error "Missing shim for ${tool} ${version}"
        ((VERIFICATION_FAILED++)) || true
    fi
done < "${TOOL_VERSIONS}"

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
