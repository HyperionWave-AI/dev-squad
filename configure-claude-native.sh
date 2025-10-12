#!/bin/bash

# Configure Claude Code to use Hyperion native binary
# This script sets up the native binary for MCP stdio mode

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
BINARY_PATH="$PROJECT_ROOT/bin/hyper"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Hyperion Native - Claude Code Configuration              ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Native binary not found at $BINARY_PATH${NC}"
    echo ""
    echo -e "Build the binary first:"
    echo -e "  ${YELLOW}make native${NC}"
    echo ""
    exit 1
fi

# Check if binary is executable
if [ ! -x "$BINARY_PATH" ]; then
    echo -e "${YELLOW}Making binary executable...${NC}"
    chmod +x "$BINARY_PATH"
fi

# Check if .env.native exists
if [ ! -f "$PROJECT_ROOT/.env.native" ]; then
    echo -e "${YELLOW}Warning: .env.native not found${NC}"
    echo -e "The binary will use system environment variables."
    echo -e "To create configuration file:"
    echo -e "  ${YELLOW}cp $PROJECT_ROOT/.env.native $PROJECT_ROOT/.env.native${NC}"
    echo ""
fi

# Remove old configuration
echo -e "${BLUE}[1/3] Removing old configuration...${NC}"
claude mcp remove hyper --scope user 2>/dev/null || true
claude mcp remove hyper --scope project 2>/dev/null || true
echo -e "${GREEN}✓ Old configuration removed${NC}"
echo ""

# Add new configuration
echo -e "${BLUE}[2/3] Adding native binary to Claude Code...${NC}"
claude mcp add hyper "$BINARY_PATH" --args "--mode=mcp" --scope user
echo -e "${GREEN}✓ Native binary configured${NC}"
echo ""

# Verify configuration
echo -e "${BLUE}[3/3] Verifying configuration...${NC}"
if claude mcp list 2>&1 | grep -q "hyper"; then
    echo -e "${GREEN}✓ Configuration verified${NC}"
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Configuration complete! ✓                                ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Binary path:  ${YELLOW}$BINARY_PATH${NC}"
    echo -e "Mode:         ${YELLOW}stdio (MCP protocol)${NC}"
    echo -e "Scope:        ${YELLOW}user (available globally)${NC}"
    echo -e "Config file:  ${YELLOW}$PROJECT_ROOT/.env.native${NC}"
    echo ""
    echo -e "${BLUE}Configuration details:${NC}"
    claude mcp list | grep hyper
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo ""
    echo -e "1. Ensure .env.native is configured:"
    echo -e "   ${YELLOW}vim $PROJECT_ROOT/.env.native${NC}"
    echo ""
    echo -e "2. Test the connection in Claude Code:"
    echo -e "   Use the hyper MCP tools in your conversations"
    echo ""
    echo -e "3. To install globally (optional):"
    echo -e "   ${YELLOW}sudo mv $BINARY_PATH /usr/local/bin/${NC}"
    echo -e "   ${YELLOW}sudo cp $PROJECT_ROOT/.env.native /usr/local/bin/.env.native${NC}"
    echo -e "   ${YELLOW}claude mcp add hyper /usr/local/bin/hyper --args \"--mode=mcp\" --scope user${NC}"
    echo ""
else
    echo -e "${RED}✗ Configuration failed${NC}"
    echo ""
    echo -e "Please check:"
    echo -e "- Claude Code CLI is installed"
    echo -e "- Binary is executable: ${YELLOW}chmod +x $BINARY_PATH${NC}"
    echo -e "- .env.native is configured with MongoDB URI"
    echo ""
    exit 1
fi
