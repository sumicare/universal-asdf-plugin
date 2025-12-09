#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.." || exit 1

echo "==> Building universal-asdf-plugin..."

mkdir -p ./build

# Build with optimizations
go build -ldflags="-s -w" -v -o ./build/universal-asdf-plugin .

echo "==> Build complete: ./build/universal-asdf-plugin"
ls -lh ./build/universal-asdf-plugin
