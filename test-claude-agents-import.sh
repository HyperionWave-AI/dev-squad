#!/bin/bash
# Test script for Claude agents import feature

echo "=== Testing Claude Agents Import Feature ==="
echo ""

# Check if coordinator is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "❌ Coordinator is not running on port 8080"
    echo "   Start it with: cd coordinator && make run"
    exit 1
fi

echo "✅ Coordinator is running"
echo ""

# Test 1: List available Claude agents
echo "Test 1: GET /api/v1/ai/claude-agents (list available agents)"
AGENTS_RESPONSE=$(curl -s http://localhost:8080/api/v1/ai/claude-agents)
AGENT_COUNT=$(echo "$AGENTS_RESPONSE" | jq -r '.count // 0')

if [ "$AGENT_COUNT" -gt 0 ]; then
    echo "✅ Found $AGENT_COUNT Claude agents"
    echo "   Sample agents:"
    echo "$AGENTS_RESPONSE" | jq -r '.agents[0:3] | .[] | "   - \(.name): \(.description)"'
else
    echo "❌ No Claude agents found"
    echo "   Response: $AGENTS_RESPONSE"
fi
echo ""

# Test 2: Import Claude agents (requires JWT token)
echo "Test 2: POST /api/v1/ai/subagents/import-claude (bulk import)"
echo "   Note: This requires JWT authentication"

# Check if JWT_TOKEN is set
if [ -z "$JWT_TOKEN" ]; then
    echo "⚠️  JWT_TOKEN not set. Generate one with:"
    echo "   node /Users/maxmednikov/MaxSpace/Hyperion/scripts/generate_jwt_50years.js"
    echo "   export JWT_TOKEN=<token>"
    echo ""
    echo "   Skipping authenticated test."
else
    # Try to import first 2 agents
    AGENT_NAMES=$(echo "$AGENTS_RESPONSE" | jq -r '.agents[0:2] | [.[].name] | @json')

    echo "   Attempting to import: $AGENT_NAMES"

    IMPORT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/ai/subagents/import-claude \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{\"agentNames\": $AGENT_NAMES}")

    IMPORTED=$(echo "$IMPORT_RESPONSE" | jq -r '.imported // 0')
    SUCCESS=$(echo "$IMPORT_RESPONSE" | jq -r '.success // false')

    if [ "$SUCCESS" = "true" ]; then
        echo "✅ Successfully imported $IMPORTED agents"
    else
        echo "⚠️  Partially imported $IMPORTED agents"
        echo "   Errors:"
        echo "$IMPORT_RESPONSE" | jq -r '.errors[]? | "   - \(.)"'
    fi

    echo ""
    echo "   Full response:"
    echo "$IMPORT_RESPONSE" | jq '.'
fi

echo ""
echo "=== Test Complete ==="
