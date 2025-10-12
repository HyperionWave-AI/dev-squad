#!/bin/bash

# Hyperion Coordinator MCP - One-Click Installer
# Automatically detects platform and installs the MCP server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Hyperion Coordinator MCP - Installer v1.0.0     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Platform detected: ${OS}-${ARCH}${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install via npm (preferred method)
install_via_npm() {
    echo -e "${YELLOW}📦 Installing via npm...${NC}"

    if ! command_exists npm; then
        echo -e "${RED}❌ npm not found${NC}"
        echo -e "Please install Node.js from: ${BLUE}https://nodejs.org/${NC}"
        return 1
    fi

    echo "Running: npm install -g @hyperion/coordinator-mcp"
    npm install -g @hyperion/coordinator-mcp

    echo -e "${GREEN}✅ Installed via npm${NC}"
    return 0
}

# Function to download and install binary
install_binary() {
    echo -e "${YELLOW}📥 Downloading pre-built binary...${NC}"

    BINARY_NAME="hyperion-coordinator-mcp"
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    # GitHub release URL (update with actual repository)
    RELEASE_URL="https://github.com/yourorg/hyperion-coordinator-mcp/releases/latest/download/hyperion-coordinator-mcp-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        RELEASE_URL="${RELEASE_URL}.exe"
    fi

    # Download directory
    if [ "$OS" = "darwin" ] || [ "$OS" = "linux" ]; then
        INSTALL_DIR="/usr/local/bin"
        INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"
    else
        INSTALL_DIR="$HOME/.local/bin"
        INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"
        mkdir -p "$INSTALL_DIR"
    fi

    echo "Downloading from: $RELEASE_URL"
    echo "Installing to: $INSTALL_PATH"

    # Download with curl or wget
    if command_exists curl; then
        curl -L "$RELEASE_URL" -o "$INSTALL_PATH"
    elif command_exists wget; then
        wget "$RELEASE_URL" -O "$INSTALL_PATH"
    else
        echo -e "${RED}❌ Neither curl nor wget found${NC}"
        echo "Please install curl or wget and try again"
        return 1
    fi

    # Make executable
    chmod +x "$INSTALL_PATH"

    echo -e "${GREEN}✅ Binary installed to $INSTALL_PATH${NC}"
    return 0
}

# Function to build from source
build_from_source() {
    echo -e "${YELLOW}🔨 Building from source...${NC}"

    if ! command_exists go; then
        echo -e "${RED}❌ Go not found${NC}"
        echo -e "Please install Go 1.25+ from: ${BLUE}https://go.dev/dl/${NC}"
        return 1
    fi

    echo "Checking Go version..."
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Go version: $GO_VERSION"

    echo "Building binary..."
    go build -o hyperion-coordinator-mcp main.go

    # Make executable
    chmod +x hyperion-coordinator-mcp

    # Move to bin directory
    if [ "$OS" = "darwin" ] || [ "$OS" = "linux" ]; then
        echo "Installing to /usr/local/bin (requires sudo)..."
        sudo mv hyperion-coordinator-mcp /usr/local/bin/
    else
        mkdir -p "$HOME/.local/bin"
        mv hyperion-coordinator-mcp "$HOME/.local/bin/"
    fi

    echo -e "${GREEN}✅ Built and installed from source${NC}"
    return 0
}

# Function to configure Claude Code
configure_claude_code() {
    echo ""
    echo -e "${YELLOW}⚙️  Configuring Claude Code...${NC}"

    # Determine Claude Code config directory
    if [ "$OS" = "darwin" ]; then
        CONFIG_DIR="$HOME/Library/Application Support/Claude"
    elif [ "$OS" = "linux" ]; then
        CONFIG_DIR="$HOME/.config/Claude"
    else
        CONFIG_DIR="$APPDATA/Claude"
    fi

    CONFIG_FILE="${CONFIG_DIR}/claude_desktop_config.json"

    # Create config directory if it doesn't exist
    mkdir -p "$CONFIG_DIR"

    # Determine binary path
    if command_exists hyperion-coordinator-mcp; then
        BINARY_PATH=$(which hyperion-coordinator-mcp)
    else
        BINARY_PATH="/usr/local/bin/hyperion-coordinator-mcp"
    fi

    # Create or update config
    if [ -f "$CONFIG_FILE" ]; then
        echo "Updating existing Claude Code configuration..."
        # Use Python/Node to safely update JSON (if available)
        if command_exists python3; then
            python3 - <<EOF
import json
import sys

config_file = "$CONFIG_FILE"
try:
    with open(config_file, 'r') as f:
        config = json.load(f)
except:
    config = {}

if 'mcpServers' not in config:
    config['mcpServers'] = {}

config['mcpServers']['hyperion-coordinator'] = {
    'command': '$BINARY_PATH'
}

with open(config_file, 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Configuration updated")
EOF
        else
            echo -e "${YELLOW}⚠️  Manual configuration required${NC}"
            echo "Add this to $CONFIG_FILE:"
            echo ""
            cat <<EOF
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "$BINARY_PATH"
    }
  }
}
EOF
            return 1
        fi
    else
        echo "Creating new Claude Code configuration..."
        cat > "$CONFIG_FILE" <<EOF
{
  "mcpServers": {
    "hyperion-coordinator": {
      "command": "$BINARY_PATH",
      "env": {
        "MONGODB_URI": "mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB",
        "MONGODB_DATABASE": "coordinator_db"
      }
    }
  }
}
EOF
    fi

    echo -e "${GREEN}✅ Claude Code configured${NC}"
    echo "Config file: $CONFIG_FILE"
    return 0
}

# Main installation logic
main() {
    echo "Choose installation method:"
    echo "1) npm (recommended - auto-updates available)"
    echo "2) Pre-built binary (no dependencies)"
    echo "3) Build from source (requires Go)"
    echo ""
    read -p "Enter choice [1-3]: " choice

    case $choice in
        1)
            if install_via_npm; then
                echo ""
                echo -e "${GREEN}🎉 Installation complete!${NC}"
                echo ""
                echo "Next steps:"
                echo "1. Restart Claude Code"
                echo "2. Verify MCP server appears in settings"
                echo "3. Test with: mcp__hyperion-coordinator__coordinator_list_human_tasks({})"
                return 0
            else
                echo -e "${YELLOW}⚠️  npm installation failed, trying binary download...${NC}"
                install_binary || build_from_source
            fi
            ;;
        2)
            install_binary || build_from_source
            ;;
        3)
            build_from_source || install_binary
            ;;
        *)
            echo -e "${RED}❌ Invalid choice${NC}"
            exit 1
            ;;
    esac

    # Configure Claude Code
    configure_claude_code

    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║          Installation Complete! 🎉                 ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Restart Claude Code to load the MCP server"
    echo "2. Verify it appears in Claude Code's MCP servers list"
    echo "3. Test with:"
    echo -e "   ${BLUE}mcp__hyperion-coordinator__coordinator_list_human_tasks({})${NC}"
    echo ""
    echo "Documentation: https://github.com/yourorg/hyperion-coordinator-mcp"
    echo "Issues: https://github.com/yourorg/hyperion-coordinator-mcp/issues"
}

# Run main function
main
