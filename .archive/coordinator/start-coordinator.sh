#!/bin/bash

# Hyperion Unified Coordinator Start Script
# Starts the unified coordinator service (REST API + UI serving)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load environment variables from parent .env file
if [ -f "../.env" ]; then
  echo "📝 Loading environment variables from ../.env"
  export $(grep -v '^#' ../.env | xargs)
fi

echo "🚀 Starting Hyperion Unified Coordinator..."
echo ""
echo "Architecture: UI → REST API → Storage Layer"
echo ""

# Check if binary exists
if [ ! -f "bin/coordinator" ]; then
  echo "❌ Coordinator binary not found. Building..."
  go build -o bin/coordinator ./cmd/coordinator
  echo "✅ Coordinator built ($(du -h bin/coordinator | cut -f1))"
fi

# Check if UI dist exists
if [ ! -d "ui/dist" ]; then
  echo "❌ UI build not found. Building..."
  cd ui
  npm install
  npm run build
  cd ..
  echo "✅ UI built"
fi

echo ""
echo "🎬 Starting unified coordinator..."
echo ""
echo "📍 Service URLs:"
echo "  - REST API:     http://localhost:7095/api/*"
echo "  - UI:           http://localhost:7095/ (served by coordinator)"
echo "  - Health:       http://localhost:7095/health"
echo "  - MongoDB:      ${MONGODB_URI:-MongoDB Atlas}"
echo "  - Qdrant:       ${QDRANT_URL:-Cloud Qdrant}"
echo ""
echo "⚠️  Press Ctrl+C to stop"
echo ""

# Function to cleanup
cleanup() {
  echo ""
  echo "🛑 Shutting down coordinator..."
  kill $COORDINATOR_PID 2>/dev/null || true
  wait
  echo "✅ Coordinator stopped"
  exit 0
}

trap cleanup INT TERM

# Start unified coordinator in HTTP mode
echo "🌟 Starting coordinator..."
./bin/coordinator --mode=http &
COORDINATOR_PID=$!

# Wait for coordinator to start
sleep 3

# Check if coordinator is healthy
if curl -s http://localhost:7095/health > /dev/null 2>&1; then
  echo "✅ Coordinator running (PID: $COORDINATOR_PID)"
else
  echo "❌ Coordinator failed to start"
  echo ""
  echo "Check logs for errors:"
  echo "  tail -f logs/coordinator.log"
  kill $COORDINATOR_PID 2>/dev/null || true
  exit 1
fi

echo ""
echo "✅ Coordinator started successfully!"
echo ""
echo "📖 Open your browser to: http://localhost:7095"
echo ""
echo "🔍 Test the REST API:"
echo "   curl http://localhost:7095/api/tasks"
echo ""
echo "💡 Create a test task:"
echo "   curl -X POST http://localhost:7095/api/tasks \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"prompt\":\"Test task from REST API\"}'"
echo ""
echo "🔎 Search code (after indexing):"
echo "   curl -X POST http://localhost:7095/api/code-index/search \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"query\":\"JWT authentication\",\"limit\":5}'"
echo ""

# Wait for process
wait
