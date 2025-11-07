#!/bin/bash

# Hyper - Native macOS Run Script
# Loads environment and runs the native binary

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
BINARY="$PROJECT_ROOT/bin/hyper"
ENV_FILE="$PROJECT_ROOT/.env.hyper"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Hyper - Native macOS                                     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}ERROR: Binary not found: $BINARY${NC}"
    echo -e "${YELLOW}Run ./build-native.sh first to build the binary${NC}"
    exit 1
fi

# Load environment variables
if [ -f "$ENV_FILE" ]; then
    echo -e "${GREEN}Loading environment from .env.hyper...${NC}"
    source "$ENV_FILE"
    echo ""
else
    echo -e "${YELLOW}Warning: .env.hyper not found${NC}"
    echo -e "${YELLOW}Make sure environment variables are set manually${NC}"
    echo ""
fi

# Run the binary
echo -e "${GREEN}Starting Hyper...${NC}"
echo -e "  Binary: $BINARY"
echo -e "  Mode: HTTP (REST API + UI)"
echo -e "  Port: ${HTTP_PORT:-7095}"
echo ""
echo -e "${BLUE}Press Ctrl+C to stop${NC}"
echo ""

# Run in HTTP mode (REST API + embedded UI)
exec "$BINARY" --mode=http
