#!/bin/bash

# Native development mode - runs both UI (Vite) and Coordinator (Air) without Docker
# Usage: ./scripts/dev-native.sh or make dev

set -e

echo "🔥 Starting Hyperion Coordinator in native dev mode..."
echo ""

# Trap SIGINT/SIGTERM to kill child processes
cleanup() {
    echo ""
    echo "🛑 Shutting down dev services..."
    if [ ! -z "$UI_PID" ]; then
        kill $UI_PID 2>/dev/null || true
    fi
    if [ ! -z "$COORDINATOR_PID" ]; then
        kill $COORDINATOR_PID 2>/dev/null || true
    fi
    wait 2>/dev/null || true
    echo "✅ Dev services stopped"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Start UI dev server (Vite with hot reload)
echo "🎨 Starting UI dev server (Vite)..."
cd ui
npm run dev &
UI_PID=$!
cd ..
echo "   → UI server starting (PID: $UI_PID)"
echo "   → Will be available at http://localhost:5173"
echo ""

# Wait for UI to start
sleep 2

# Start Coordinator with Air hot reload
echo "🔨 Starting Coordinator with Air hot reload..."
air &
COORDINATOR_PID=$!
echo "   → Coordinator starting (PID: $COORDINATOR_PID)"
echo "   → API: http://localhost:7095/api"
echo "   → UI (proxied): http://localhost:7095/ui"
echo ""

echo "✅ Development mode active!"
echo ""
echo "📖 URLs:"
echo "   • UI (direct):   http://localhost:5173"
echo "   • UI (proxied):  http://localhost:7095/ui  ← Use this for API access"
echo "   • API:           http://localhost:7095/api"
echo "   • Health:        http://localhost:7095/health"
echo ""
echo "🔥 Hot reload enabled for both UI and Coordinator"
echo "Press Ctrl+C to stop all services"
echo ""

# Wait for both processes
wait $UI_PID $COORDINATOR_PID
