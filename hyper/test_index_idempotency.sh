#!/bin/bash
# Test script to verify MongoDB index creation idempotency

set -e

echo "=== Testing MongoDB Index Idempotency ==="
echo ""

# Set required environment variables
export MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
export MONGODB_DATABASE="${MONGODB_DATABASE:-hyper_hyperion_test}"
export QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
export LOG_LEVEL="${LOG_LEVEL:-info}"

# Build the coordinator
echo "Building coordinator..."
make build-hyper > /dev/null 2>&1

# Function to start and stop coordinator
test_startup() {
    local attempt=$1
    echo "Attempt $attempt: Starting coordinator..."

    # Start coordinator in background
    ./bin/hyper > /tmp/hyper_test_$attempt.log 2>&1 &
    HYPER_PID=$!

    # Wait for startup (5 seconds)
    sleep 5

    # Check if process is still running (didn't crash)
    if kill -0 $HYPER_PID 2>/dev/null; then
        echo "✅ Attempt $attempt: Coordinator started successfully"

        # Check logs for index errors
        if grep -q "failed to create.*index" /tmp/hyper_test_$attempt.log; then
            echo "❌ Attempt $attempt: Index creation errors detected!"
            grep "failed to create.*index" /tmp/hyper_test_$attempt.log
            kill $HYPER_PID 2>/dev/null || true
            return 1
        fi

        # Stop coordinator
        kill $HYPER_PID 2>/dev/null || true
        sleep 2
        echo "✅ Attempt $attempt: No index errors detected"
        return 0
    else
        echo "❌ Attempt $attempt: Coordinator failed to start!"
        cat /tmp/hyper_test_$attempt.log
        return 1
    fi
}

# Test multiple startups
echo ""
echo "Testing multiple coordinator startups..."
echo ""

success_count=0
for i in 1 2 3; do
    if test_startup $i; then
        ((success_count++))
    else
        echo ""
        echo "❌ TEST FAILED on attempt $i"
        exit 1
    fi
    echo ""
done

echo "=== IDEMPOTENCY TEST PASSED ==="
echo "Successfully started coordinator $success_count times without index errors"
echo ""
echo "Log files available at: /tmp/hyper_test_*.log"
