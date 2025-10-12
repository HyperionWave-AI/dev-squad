#!/bin/bash

# Test script for port-busy recovery feature
# This script demonstrates the interactive prompt when the HTTP server port is already in use

set -e

PORT=7095

echo "=== Port-Busy Recovery Test ==="
echo ""
echo "This test demonstrates the interactive port-busy recovery feature."
echo "When the HTTP server detects the port is busy, it will:"
echo "  1. Find the process using the port"
echo "  2. Prompt you to kill it"
echo "  3. Automatically retry starting the server"
echo ""

# Check if port is already in use
if lsof -ti tcp:$PORT > /dev/null 2>&1; then
    PID=$(lsof -ti tcp:$PORT | head -n1)
    echo "⚠️  Port $PORT is already in use by process $PID"
    echo "Killing existing process..."
    kill $PID 2>/dev/null || true
    sleep 1
fi

echo "Step 1: Starting a dummy process on port $PORT..."
# Start a simple HTTP server to block the port
python3 -c "import http.server; http.server.HTTPServer(('', $PORT), http.server.SimpleHTTPRequestHandler).serve_forever()" &
BLOCKER_PID=$!

# Wait for the blocker to start
sleep 2

echo "Step 2: Dummy process started (PID: $BLOCKER_PID)"
echo ""

# Find actual PID using the port (may be different on some systems)
ACTUAL_PID=$(lsof -ti tcp:$PORT | head -n1)
echo "Process using port $PORT: $ACTUAL_PID"
echo ""

echo "Step 3: Now try to start the hyper HTTP server..."
echo "The server will detect the port is busy and prompt you to kill the process."
echo ""
echo "Press Enter to continue..."
read

# Try to start the hyper server
# It should detect the port is busy and prompt to kill the process
echo ""
echo "Starting hyper HTTP server..."
echo "You should see a prompt like:"
echo "  ⚠️  Port $PORT is already in use by process $ACTUAL_PID"
echo "  Kill the process and retry? [y/N]:"
echo ""
echo "Type 'y' to kill the blocking process and retry."
echo ""

# Note: We can't easily test the interactive prompt in a script
# This script is meant to be run manually to demonstrate the feature

# Clean up
echo ""
echo "Cleaning up..."
kill $BLOCKER_PID 2>/dev/null || true

echo ""
echo "=== Test Complete ==="
echo ""
echo "To fully test the interactive prompt:"
echo "1. Start a process on port $PORT:"
echo "   python3 -m http.server $PORT"
echo ""
echo "2. In another terminal, start hyper:"
echo "   ./bin/hyper --mode=http"
echo ""
echo "3. You should see the interactive prompt asking to kill the process."
echo ""
