#!/bin/bash

# Hyperion Coordinator Full Stack Start Script
# Starts HTTP Bridge (which starts MCP Server) and React UI

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting Hyperion Coordinator Full Stack..."
echo ""

# Check if binaries exist
if [ ! -f "mcp-server/hyperion-coordinator-mcp" ]; then
  echo "❌ MCP server binary not found. Building..."
  cd mcp-server
  go build -o hyperion-coordinator-mcp
  cd ..
  echo "✅ MCP server built"
fi

if [ ! -f "mcp-http-bridge/hyperion-coordinator-bridge" ]; then
  echo "❌ HTTP bridge binary not found. Building..."
  cd mcp-http-bridge
  go build -o hyperion-coordinator-bridge
  cd ..
  echo "✅ HTTP bridge built"
fi

# Check if UI dependencies are installed
if [ ! -d "ui/node_modules" ]; then
  echo "📦 Installing UI dependencies..."
  cd ui
  npm install
  cd ..
  echo "✅ UI dependencies installed"
fi

echo ""
echo "🎬 Starting services..."
echo ""
echo "📍 Service URLs:"
echo "  - HTTP Bridge:  http://localhost:7095"
echo "  - React UI:     http://localhost:5173"
echo "  - MongoDB:      MongoDB Atlas (coordinator_db)"
echo ""
echo "⚠️  Press Ctrl+C to stop all services"
echo ""

# Function to cleanup background processes
cleanup() {
  echo ""
  echo "🛑 Shutting down services..."
  kill $BRIDGE_PID 2>/dev/null || true
  kill $UI_PID 2>/dev/null || true
  wait
  echo "✅ All services stopped"
  exit 0
}

trap cleanup INT TERM

# Start HTTP Bridge (which will start MCP server)
echo "🌉 Starting HTTP Bridge..."
cd mcp-http-bridge
./hyperion-coordinator-bridge &
BRIDGE_PID=$!
cd ..

# Wait for bridge to start
sleep 2

# Check if bridge is healthy
if curl -s http://localhost:7095/health > /dev/null 2>&1; then
  echo "✅ HTTP Bridge running (PID: $BRIDGE_PID)"
else
  echo "❌ HTTP Bridge failed to start"
  kill $BRIDGE_PID 2>/dev/null || true
  exit 1
fi

# Start React UI
echo "🎨 Starting React UI..."
cd ui
npm run dev &
UI_PID=$!
cd ..

# Wait for UI to start
sleep 3

echo ""
echo "✅ All services started!"
echo ""
echo "📖 Open your browser to: http://localhost:5173"
echo ""
echo "🔍 Test the HTTP bridge:"
echo "   curl http://localhost:7095/health"
echo ""
echo "💡 Create a test task:"
echo "   curl -X POST http://localhost:7095/api/mcp/tools/call \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -H 'X-Request-ID: test-1' \\"
echo "     -d '{\"name\":\"coordinator_create_human_task\",\"arguments\":{\"prompt\":\"Test task\"}}'"
echo ""

# Wait for processes
wait