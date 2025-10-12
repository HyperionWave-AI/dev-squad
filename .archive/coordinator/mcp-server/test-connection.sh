#!/bin/bash

# Test script to verify MongoDB Atlas connection for Hyperion Coordinator MCP

echo "======================================"
echo "Testing Coordinator MCP MongoDB Setup"
echo "======================================"
echo ""

# Check if binary exists
if [ ! -f "./hyperion-coordinator-mcp" ]; then
    echo "❌ Binary not found. Building..."
    go build -o hyperion-coordinator-mcp || {
        echo "❌ Build failed"
        exit 1
    }
    echo "✅ Build successful"
fi

echo "Starting coordinator MCP server..."
echo "Press Ctrl+C after you see 'Successfully connected to MongoDB Atlas'"
echo ""

# Run the server and capture output
timeout 5 ./hyperion-coordinator-mcp 2>&1 || true

echo ""
echo "======================================"
echo "MongoDB Connection Test Complete"
echo "======================================"
echo ""
echo "If you saw 'Successfully connected to MongoDB Atlas', the setup is working!"
echo ""
echo "Next steps:"
echo "1. Add to your MCP client config (Claude Desktop, etc.)"
echo "2. Use the 5 coordinator tools to manage tasks and knowledge"
echo "3. Check MongoDB Atlas to see persisted data"