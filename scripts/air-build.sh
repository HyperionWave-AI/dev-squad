#!/bin/bash
# Air Build Script for Native Binary
# Called by Air during hot reload - keeps build fast

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

# Check if HOT_RELOAD mode is enabled
if [ "$HOT_RELOAD" = "true" ]; then
  echo "🔥 Hot-reload mode: Building unified hyper WITHOUT embedded UI (will proxy to Vite)"

  # Build unified hyper binary WITHOUT embedded UI (no -tags embed)
  cd hyper/cmd/coordinator
  go build \
    -ldflags="-s -w -X main.Version=dev-hot-reload -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ') -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
    -o ../../../bin/hyper \
    .

  echo "✓ Unified hyper binary rebuilt (hot-reload mode): $(ls -lh ../../../bin/hyper | awk '{print $5}')"
else
  echo "📦 Production mode: Building unified hyper WITH embedded UI"

  # Copy UI to embed directory (Go embed doesn't support symlinks)
  rm -rf hyper/embed/ui
  mkdir -p hyper/embed/ui
  cp -r coordinator/ui/dist hyper/embed/ui/

  # Build unified hyper binary with embedded UI
  cd hyper/cmd/coordinator
  go build \
    -tags embed \
    -ldflags="-s -w -X main.Version=dev-hot-reload -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ') -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
    -o ../../../bin/hyper \
    .

  echo "✓ Unified hyper binary rebuilt (production mode): $(ls -lh ../../../bin/hyper | awk '{print $5}')"
fi
