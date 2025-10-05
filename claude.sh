#!/bin/bash

# Claude Code Interactive Launcher with Devcontainer Support
# This script launches Claude Code with the --dangerously-skip-permissions flag
# for use in devcontainer environments

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Check if running in a devcontainer
is_devcontainer() {
    if [ -f "/.dockerenv" ] || [ -n "${REMOTE_CONTAINERS}" ] || [ -n "${CODESPACES}" ]; then
        return 0
    fi
    return 1
}

# Main script
print_color "$BLUE" "🚀 Claude Code Interactive Launcher"
print_color "$BLUE" "===================================="
echo

# Check environment
if is_devcontainer; then
    print_color "$GREEN" "✓ Detected devcontainer environment"
    USE_DANGEROUS_FLAG=true
else
    print_color "$YELLOW" "⚠️  Not running in a devcontainer"
    echo
    read -p "Do you want to use --dangerously-skip-permissions anyway? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        USE_DANGEROUS_FLAG=true
        print_color "$YELLOW" "⚠️  Using --dangerously-skip-permissions flag"
    else
        USE_DANGEROUS_FLAG=false
        print_color "$GREEN" "✓ Running with normal permissions"
    fi
fi

echo

# Build the command
CLAUDE_CMD="~/.claude/local/claude"

# Add the dangerous flag if needed
if [ "$USE_DANGEROUS_FLAG" = true ]; then
    CLAUDE_CMD="$CLAUDE_CMD --dangerously-skip-permissions"
fi

# Check for additional arguments
if [ $# -gt 0 ]; then
    print_color "$BLUE" "📝 Additional arguments detected: $*"
    CLAUDE_CMD="$CLAUDE_CMD $*"
fi

# Display the command to be executed
print_color "$GREEN" "▶️  Executing: $CLAUDE_CMD"
echo

# Execute Claude Code
exec $CLAUDE_CMD