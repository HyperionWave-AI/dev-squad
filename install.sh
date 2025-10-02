#!/bin/bash

# Hyperion Coordinator MCP - Docker Installation Script
# This script builds and configures the MCP server for use with Claude Code or other MCP clients

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Check prerequisites
check_docker() {
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed. Please install Docker first:"
        echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
        echo "  Linux: https://docs.docker.com/engine/install/"
        echo "  Windows: https://docs.docker.com/desktop/install/windows-install/"
        exit 1
    fi
    success "Docker is installed"
}

check_docker_compose() {
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        error "Docker Compose is not installed"
        exit 1
    fi
    success "Docker Compose is installed"
}

# Main installation
main() {
    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo "  Hyperion Coordinator MCP - Docker Installation"
    echo "═══════════════════════════════════════════════════════"
    echo ""

    # Check prerequisites
    info "Checking prerequisites..."
    check_docker
    check_docker_compose
    echo ""

    # Create .env if it doesn't exist
    if [ ! -f .env ]; then
        info "Creating .env file from template..."
        cp .env.example .env
        success ".env file created (using default MongoDB Atlas dev cluster)"
    else
        warning ".env file already exists, skipping creation"
    fi
    echo ""

    # Build the Docker image
    info "Building Hyperion Coordinator MCP Docker image..."
    docker-compose build
    success "Docker image built successfully"
    echo ""

    # Get the absolute path to the docker-compose file
    COMPOSE_FILE="$(cd "$(dirname "$0")" && pwd)/docker-compose.yml"

    # Detect Claude Code config location
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        CLAUDE_CONFIG="$HOME/Library/Application Support/Claude/claude_desktop_config.json"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        CLAUDE_CONFIG="$HOME/.config/Claude/claude_desktop_config.json"
    else
        # Unsupported OS
        warning "Unsupported OS for automatic Claude Code configuration"
        CLAUDE_CONFIG=""
    fi

    # Configure Claude Code if config file exists
    if [ -n "$CLAUDE_CONFIG" ] && [ -f "$CLAUDE_CONFIG" ]; then
        info "Configuring Claude Code..."

        # Create backup
        cp "$CLAUDE_CONFIG" "$CLAUDE_CONFIG.backup"

        # Check if hyperion-coordinator already exists
        if grep -q '"hyperion-coordinator"' "$CLAUDE_CONFIG"; then
            warning "hyperion-coordinator already configured in Claude Code"
        else
            # Add the MCP server configuration
            # Using docker exec to connect to running container
            cat <<EOF | python3 - "$CLAUDE_CONFIG"
import json
import sys

config_file = sys.argv[1]
with open(config_file, 'r') as f:
    config = json.load(f)

if 'mcpServers' not in config:
    config['mcpServers'] = {}

config['mcpServers']['hyperion-coordinator'] = {
    'command': '/usr/local/bin/docker',
    'args': [
        'exec',
        '-i',
        'hyperion-http-bridge',
        '/app/hyperion-coordinator-mcp'
    ],
    'env': {}
}

with open(config_file, 'w') as f:
    json.dump(config, f, indent=2)
EOF
            success "Claude Code configured successfully"
            info "Restart Claude Code to load the MCP server"
        fi
    else
        warning "Claude Code config not found at: $CLAUDE_CONFIG"
        info "Manual configuration required (see below)"
    fi
    echo ""

    # Success message
    echo "═══════════════════════════════════════════════════════"
    success "Installation complete!"
    echo "═══════════════════════════════════════════════════════"
    echo ""

    # Show next steps
    echo "📋 Next Steps:"
    echo ""
    echo "1. Start the MCP server:"
    echo "   ${GREEN}docker-compose up -d${NC}"
    echo ""
    echo "2. View logs:"
    echo "   ${GREEN}docker-compose logs -f hyperion-coordinator-mcp${NC}"
    echo ""
    echo "3. Stop the server:"
    echo "   ${GREEN}docker-compose down${NC}"
    echo ""

    if [ -z "$CLAUDE_CONFIG" ] || [ ! -f "$CLAUDE_CONFIG" ]; then
        echo "4. Manual Claude Code Configuration:"
        echo "   Add this to your Claude Code config file:"
        echo "   ${BLUE}Location: $CLAUDE_CONFIG${NC}"
        echo ""
        echo '   {'
        echo '     "mcpServers": {'
        echo '       "hyperion-coordinator": {'
        echo '         "type": "stdio",'
        echo '         "command": "/usr/local/bin/docker",'
        echo '         "args": ['
        echo '           "exec",'
        echo '           "-i",'
        echo '           "hyperion-http-bridge",'
        echo '           "/app/hyperion-coordinator-mcp"'
        echo '         ],'
        echo '         "env": {}'
        echo '       }'
        echo '     }'
        echo '   }'
        echo ""
    fi

    echo "📚 Documentation:"
    echo "   - README: ./coordinator/mcp-server/README.md"
    echo "   - Tool Reference: ./HYPERION_COORDINATOR_MCP_REFERENCE.md"
    echo "   - Agent Guide: ./CLAUDE.md"
    echo ""
    echo "🔧 Configuration:"
    echo "   - Edit .env to customize MongoDB settings"
    echo "   - Default: Uses dev MongoDB Atlas cluster"
    echo ""
}

# Run main function
main
