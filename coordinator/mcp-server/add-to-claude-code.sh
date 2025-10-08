#!/bin/bash

# Script to add Hyperion Coordinator MCP to Claude Code

set -e

echo "======================================"
echo "Adding Coordinator MCP to Claude Code"
echo "======================================"
echo ""

# Get absolute path to the binary
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BINARY_PATH="$SCRIPT_DIR/hyper-mcp"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ Binary not found at: $BINARY_PATH"
    echo ""
    echo "Building the binary first..."
    cd "$SCRIPT_DIR"
    go build -o hyper-mcp || {
        echo "❌ Build failed"
        exit 1
    }
    echo "✅ Build successful"
    echo ""
fi

echo "📍 Binary location: $BINARY_PATH"
echo ""

# Add to Claude Code
echo "Adding to Claude Code MCP configuration..."
echo ""

claude mcp add hyper "$BINARY_PATH" || {
    echo ""
    echo "❌ Failed to add MCP server to Claude Code"
    echo ""
    echo "Manual setup:"
    echo "1. Run: claude mcp add hyper \"$BINARY_PATH\""
    echo "2. Or edit your Claude Code config manually"
    exit 1
}

echo ""
echo "======================================"
echo "✅ Successfully Added to Claude Code"
echo "======================================"
echo ""
echo "The Hyperion Coordinator MCP server is now available!"
echo ""
echo "📦 MCP Tools (5):"
echo "  • coordinator_create_human_task"
echo "  • coordinator_create_agent_task"
echo "  • coordinator_update_task_status"
echo "  • coordinator_upsert_knowledge"
echo "  • coordinator_query_knowledge"
echo ""
echo "🔗 MCP Resources (2):"
echo "  • hyperion://task/human/{taskId}"
echo "  • hyperion://task/agent/{agentName}/{taskId}"
echo ""
echo "💾 Storage: MongoDB Atlas (coordinator_db)"
echo ""
echo "🎯 Next steps:"
echo "  1. Restart Claude Code (if running)"
echo "  2. Use coordinator tools to manage tasks and knowledge"
echo "  3. Check MongoDB Atlas to see persisted data"
echo ""