#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

COVERAGE_FILE="coverage.out"
TIMEOUT="30m"
UPDATE_SNAPSHOTS=""
RACE=""

# All tests now run in online mode by default

CPU_CORES=$(getconf _NPROCESSORS_ONLN 2>/dev/null || true)
CPU_CORES=${CPU_CORES:-""}
if [[ -z "${CPU_CORES}" ]]; then
    CPU_CORES=$(nproc 2>/dev/null || true)
fi
CPU_CORES=${CPU_CORES:-1}

# Parse flags
while [[ $# -gt 0 ]]; do
	case $1 in
		--update)
			UPDATE_SNAPSHOTS="-update"
			;;
		--race)
			RACE="-race"
			;;
		*)
			echo "Unknown option: $1"
			exit 1
			;;
	esac
	shift
done

if [[ "${ONLINE:-}" == "1" && -z "${ASDF_TEST_DOWNLOAD_CACHE_DIR:-}" ]]; then
    export ASDF_TEST_DOWNLOAD_CACHE_DIR="${HOME}/.asdf/downloads"
fi

TEST_PARALLELISM=${CPU_CORES}
GOLDIE_JOBS=${CPU_CORES}

if [[ -n "${RACE}" && -n "${UPDATE_SNAPSHOTS}" ]]; then
    GOLDIE_JOBS=1
fi

GO_TEST_FLAGS=("-timeout=${TIMEOUT}" "-p=${TEST_PARALLELISM}" "-parallel=${TEST_PARALLELISM}")
GO_TEST_FLAGS_GOLDIE=("-timeout=${TIMEOUT}" "-p=1" "-parallel=1")

if [[ -n "${RACE}" ]]; then
    GO_TEST_FLAGS+=("-race")
    GO_TEST_FLAGS_GOLDIE+=("-race")
fi

TEST_EXIT=0

if [[ -n "${UPDATE_SNAPSHOTS}" ]]; then
    go clean -testcache

    GOLDIE_PKGS=$(go list -f '{{.ImportPath}} {{.TestImports}} {{.XTestImports}}' ./... \
        | awk '/github.com\/sebdah\/goldie\/v2/ || /plugins\/asdf\/testutil/ {print $1}' \
        | sort -u || true)
    [[ -z "$GOLDIE_PKGS" ]] && { echo "No goldie packages found"; exit 0; }

    pids=()
    goldie_exit=0
    for pkg in $GOLDIE_PKGS; do
        (
            if ! go test "${GO_TEST_FLAGS_GOLDIE[@]}" "${pkg}" -run 'Goldie' ${UPDATE_SNAPSHOTS}; then
                exit 1
            fi
        ) &

        pids+=("$!")

        if (( ${#pids[@]} >= GOLDIE_JOBS )); then
            for pid in "${pids[@]}"; do
                if ! wait "${pid}"; then
                    goldie_exit=1
                fi
            done
            pids=()
        fi
    done

    for pid in "${pids[@]}"; do
        if ! wait "${pid}"; then
            goldie_exit=1
        fi
    done

    if [[ "${goldie_exit}" -ne 0 ]]; then
        TEST_EXIT=1
    fi
else
    # Run tests with coverage profile directly. This is more portable and avoids
    # go tool covdata warnings when no coverage directories are produced.
    if ! go test "${GO_TEST_FLAGS[@]}" -coverprofile="${COVERAGE_FILE}" ./...; then
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
