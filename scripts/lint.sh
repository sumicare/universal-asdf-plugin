#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

gitleaks -v detect --source .
golangci-lint run ./...
go vet ./...
go mod tidy
