#!/usr/bin/env bash
# Mutation testing with gremlins (https://gremlins.dev/)
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

# Install gremlins if needed
command -v gremlins &>/dev/null || go install github.com/go-gremlins/gremlins/cmd/gremlins@latest

# Cleanup gremlins workdirs to prevent inode exhaustion
cleanup() { find /tmp -maxdepth 1 -type d -name "gremlins-*" -exec rm -rf {} \; 2>/dev/null || true; }
trap cleanup EXIT
cleanup

PACKAGE="${1:-all}"
REPORT_DIR="./mutation-reports"
mkdir -p "${REPORT_DIR}"

run_test() {
    local pkg="$1" output
    echo "[INFO] Testing: ${pkg}"
    
    output=$(gremlins unleash --timeout-coefficient=10 \
        --exclude-files='test_helpers\.go$' \
        --exclude-files='_test\.go$' \
        "${pkg}" 2>&1) || true
    cleanup
    
    if echo "${output}" | grep -qE "panic:|ERROR:"; then
        echo "[WARN]   Skipped (gremlins error)"
    elif echo "${output}" | grep -q "No results"; then
        echo "[INFO]   No mutants"
    else
        echo "${output}" | grep -E "KILLED|LIVED|efficacy" | head -5
        echo "${output}" > "${REPORT_DIR}/$(basename "${pkg}").log"
    fi
}

if [[ "${PACKAGE}" == "all" ]]; then
    echo "[INFO] Running mutation tests on all packages..."
    for pkg in $(go list ./... | grep -vE "examples|/mock$"); do
        run_test "./${pkg#github.com/sumicare/universal-asdf-plugin/}"
    done
    
    echo ""
    echo "[INFO] Summary:"
    for f in "${REPORT_DIR}"/*.log; do
        [[ -f "$f" ]] || continue
        grep -E "Killed:|Lived:|efficacy" "$f" | head -1
        echo "  $(basename "$f" .log)"
    done
else
    run_test "${PACKAGE}"
fi

echo "[INFO] Done"
