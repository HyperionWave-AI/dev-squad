#!/bin/bash
# Native Binary Development Mode with Hot Reload
# Uses Air to watch Go and React files and auto-rebuild bin/hyper

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Unified Hyper - Native Development Mode with Hot Reload  ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Ensure GOPATH/bin is in PATH for Air
export PATH="$(go env GOPATH)/bin:$PATH"

# Check if Air is installed
if ! command -v air &> /dev/null; then
    echo -e "${RED}ERROR: Air is not installed${NC}"
    echo -e "Install with: ${YELLOW}make install-air${NC}"
    echo -e "Or manually: ${YELLOW}go install github.com/air-verse/air@latest${NC}"
    exit 1
fi

# Source environment variables from .env.hyper
if [ -f "$PROJECT_ROOT/.env.hyper" ]; then
    echo -e "${GREEN}Loading environment from .env.hyper...${NC}"
    set -a  # Export all variables
    source "$PROJECT_ROOT/.env.hyper"
    set +a
    echo -e "${GREEN}✓ Environment loaded${NC}"
else
    echo -e "${YELLOW}WARNING: .env.hyper not found${NC}"
    echo -e "  Using system environment variables"
fi

echo ""
echo -e "${BLUE}Configuration:${NC}"
echo -e "  MongoDB:    ${MONGODB_URI}"
echo -e "  Qdrant:     ${QDRANT_URL}"
echo -e "  Embedding:  ${EMBEDDING:-local}"
echo -e "  HTTP Port:  ${HTTP_PORT:-7095}"
echo ""

# Check if .air.toml exists
if [ ! -f "$PROJECT_ROOT/.air.toml" ]; then
    echo -e "${RED}ERROR: .air.toml not found at project root${NC}"
    exit 1
fi

echo -e "${BLUE}Starting Air hot reload for unified hyper binary...${NC}"
echo -e "  Watching:   hyper/**/*.go"
echo -e "  Binary:     bin/hyper (unified)"
echo -e "  Mode:       http"
echo -e "  UI:         http://localhost:${HTTP_PORT:-7095}/ui"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Stopping development server...${NC}"

    # Kill Air and all its child processes (including hyper binary)
    # This ensures proper cleanup even if Air spawned grandchildren
    pkill -P $$ 2>/dev/null || true

    # Give processes time to shutdown gracefully (Air will send SIGINT to hyper)
    sleep 1

    # Double-check: kill any remaining hyper processes
    pkill -f "bin/hyper" 2>/dev/null || true

    echo -e "${GREEN}✓ Stopped${NC}"
    exit 0
}

# Trap Ctrl+C
trap cleanup INT TERM

# Run Air with project root config
air -c "$PROJECT_ROOT/.air.toml"
