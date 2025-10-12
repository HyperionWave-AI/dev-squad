#!/bin/bash
# Air Build Script for Native Binary
# Called by Air during hot reload - keeps build fast

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

# Copy UI to embed directory (Go embed doesn't support symlinks)
rm -rf coordinator/embed/ui
mkdir -p coordinator/embed/ui
cp -r coordinator/ui/dist coordinator/embed/ui/

# Build Go binary with embedded UI
cd coordinator/cmd/coordinator
go build \
  -tags embed \
  -ldflags="-s -w -X main.Version=dev-hot-reload -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ') -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
  -o ../../../bin/hyper \
  .

echo "✓ Binary rebuilt: $(ls -lh ../../../bin/hyper | awk '{print $5}')"
