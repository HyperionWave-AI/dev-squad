#!/bin/bash
# Install Hyperion Coordinator MCP Server to Claude Code
# Uses HTTP transport with Docker container

set -e

echo "======================================"
echo "Hyperion Coordinator MCP Installation"
echo "======================================"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running"
    echo "Please start Docker Desktop and try again"
    exit 1
fi

echo "✅ Docker is running"
echo ""

# Check if container exists
if ! docker ps -a | grep -q hyperion-coordinator-mcp; then
    echo "⚠️  Container not found. Starting it now..."
    cd "$(dirname "$0")"
    docker-compose -f docker-compose.mcp-only.yml up -d
    echo "✅ Container started"
else
    # Check if container is running
    if ! docker ps | grep -q hyperion-coordinator-mcp; then
        echo "⚠️  Container exists but not running. Starting it..."
        docker start hyperion-coordinator-mcp
        echo "✅ Container started"
    else
        echo "✅ Container is running"
    fi
fi

echo ""

# Wait for health check
echo "⏳ Waiting for MCP server to be ready..."
sleep 3

# Test health endpoint
if curl -sf http://localhost:7778/health > /dev/null; then
    echo "✅ MCP server is healthy"
else
    echo "❌ MCP server health check failed"
    echo "Check logs: docker logs hyperion-coordinator-mcp"
    exit 1
fi

echo ""
echo "======================================"
echo "Adding to Claude Code"
echo "======================================"
echo ""

# Add to Claude Code using HTTP transport
echo "Running: claude mcp add --transport http hyperion-coordinator http://localhost:7778/mcp"
echo ""

claude mcp add --transport http hyperion-coordinator http://localhost:7778/mcp

echo ""
echo "======================================"
echo "✅ Installation Complete!"
echo "======================================"
echo ""
echo "📦 MCP Server: hyperion-coordinator"
echo "🌐 URL: http://localhost:7778/mcp"
echo "🚀 Transport: HTTP"
echo "📊 Tools: 9 available"
echo ""
echo "🎯 Next Steps:"
echo "  1. Restart Claude Code (if currently open)"
echo "  2. Run: /mcp"
echo "  3. Verify 'hyperion-coordinator' appears in the list"
echo "  4. Test: mcp__hyperion-coordinator__coordinator_list_human_tasks({})"
echo ""
echo "📝 Available Tools:"
echo "  • coordinator_create_human_task"
echo "  • coordinator_create_agent_task"
echo "  • coordinator_list_human_tasks"
echo "  • coordinator_list_agent_tasks"
echo "  • coordinator_update_task_status"
echo "  • coordinator_update_todo_status"
echo "  • coordinator_upsert_knowledge"
echo "  • coordinator_query_knowledge"
echo "  • coordinator_clear_task_board"
echo ""
echo "🔧 Management:"
echo "  • View logs: docker logs hyperion-coordinator-mcp -f"
echo "  • Restart: docker restart hyperion-coordinator-mcp"
echo "  • Remove: claude mcp remove hyperion-coordinator"
echo ""
