#!/usr/bin/env bash
#
# Install universal-asdf-plugin as asdf plugins
#
# Usage:
#   ./scripts/install.sh              # Install all plugins
#   ./scripts/install.sh golang rust  # Install specific plugins
#
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

# Configuration
ASDF_DATA_DIR="${ASDF_DATA_DIR:-$HOME/.asdf}"
PROJECT_DIR="$(pwd)"
BINARY="${PROJECT_DIR}/build/universal-asdf-plugin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Build if binary doesn't exist
if [[ ! -x "${BINARY}" ]]; then
    log_info "Building universal-asdf-plugin..."
    ./scripts/build.sh
fi

# Ensure asdf plugins directory exists
mkdir -p "${ASDF_DATA_DIR}/plugins"

# Install plugins
if [[ $# -gt 0 ]]; then
    log_info "Installing specified plugins: $*"
    "${BINARY}" install-plugin "$@"
else
    log_info "Installing all available plugins..."
    "${BINARY}" install-plugin
fi

log_info "Installation complete!"
log_info "Plugins installed to: ${ASDF_DATA_DIR}/plugins"
echo ""
log_info "You can now use asdf to install tools:"
echo "  asdf install golang latest"
echo "  asdf install python 3.12.0"
echo "  asdf install nodejs 20.10.0"
