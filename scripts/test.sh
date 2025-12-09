#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

COVERAGE_FILE="coverage.out"
TIMEOUT="30m"
UPDATE_SNAPSHOTS=""

for arg in "$@"; do
    case $arg in
        --online) export ONLINE=1 ;;
        --update) UPDATE_SNAPSHOTS="-update" && export ONLINE=1 ;;
        --help|-h)
            echo "Usage: $0 [--online] [--update] [--help]"
            echo "  --online  Run online/slow tests (downloads and compiles large releases, may take 5-15 min)"
            echo "  --update  Update goldie snapshots"
            exit 0
            ;;
    esac
done

TEST_EXIT=0

if [[ -n "${UPDATE_SNAPSHOTS}" ]]; then
    go clean -testcache

    GOLDIE_PKGS=$(go list -f '{{.ImportPath}} {{.TestImports}} {{.XTestImports}}' ./... \
        | awk '/github.com\/sebdah\/goldie\/v2/ || /plugins\/asdf\/testutil/ {print $1}' \
        | sort -u || true)
    [[ -z "$GOLDIE_PKGS" ]] && { echo "No goldie packages found"; exit 0; }
    for pkg in $GOLDIE_PKGS; do
        if ! go test -timeout="${TIMEOUT}" "${pkg}" -run 'Goldie' ${UPDATE_SNAPSHOTS}; then
            TEST_EXIT=1
        fi
    done
else
    # Run tests with coverage profile directly. This is more portable and avoids
    # go tool covdata warnings when no coverage directories are produced.
    if ! go test -timeout="${TIMEOUT}" -coverprofile="${COVERAGE_FILE}" ./...; then
        TEST_EXIT=$?
    fi

    # Only attempt to report coverage if the coverage file exists.
    if [[ -f "${COVERAGE_FILE}" ]]; then
        go tool cover -func="${COVERAGE_FILE}" | grep -v "100.0%" || true
    else
        echo "No coverage data generated (missing ${COVERAGE_FILE})"
    fi
fi

exit ${TEST_EXIT}
