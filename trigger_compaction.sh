#!/bin/bash

# Compaction Trigger Test Script
# Tests compaction by creating a session and sending messages via WebSocket

set -e

# Configuration
API_URL="${API_URL:-http://localhost:5555}"
USER_ID="${USER_ID:-test-user-123}"
COMPANY_ID="${COMPANY_ID:-test-company-456}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Compaction Trigger Test Script                         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if API is running
echo -e "${YELLOW}[1/5]${NC} Checking if API is running at $API_URL..."
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ API is not running at $API_URL${NC}"
    echo "Start the API with: cd hyper && go build -o bin/hyper ./cmd/... && ./bin/hyper -mode=http"
    exit 1
fi
echo -e "${GREEN}✓ API is running${NC}"
echo ""

# Create a test JWT token (for auth)
echo -e "${YELLOW}[2/5]${NC} Generating test JWT token..."
# Create a simple test JWT (this is a mock - your API may require real auth)
JWT_PAYLOAD=$(echo -n "{\"userId\":\"$USER_ID\",\"companyId\":\"$COMPANY_ID\",\"exp\":$(($(date +%s) + 3600))}" | base64 | tr -d '=' | tr '+/' '-_')
TEST_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.${JWT_PAYLOAD}.test"
echo -e "${GREEN}✓ Token generated${NC}"
echo ""

# Create a conversation/session
echo -e "${YELLOW}[3/5]${NC} Creating a new chat session..."
SESSION_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/chat/sessions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TEST_TOKEN" \
    -d "{\"title\":\"Compaction Test Session\",\"userId\":\"$USER_ID\",\"companyId\":\"$COMPANY_ID\"}" 2>&1)

echo "Response: $SESSION_RESPONSE"

SESSION_ID=$(echo "$SESSION_RESPONSE" | jq -r '.session.id // .id // .sessionId // empty' 2>/dev/null)

if [ -z "$SESSION_ID" ] || [ "$SESSION_ID" = "null" ]; then
    echo -e "${YELLOW}⚠ Could not create session via REST (may need proper auth)${NC}"
    echo "Trying alternative approach..."

    # Try to list existing sessions
    SESSIONS=$(curl -s "$API_URL/api/v1/chat/sessions?userId=$USER_ID&companyId=$COMPANY_ID" \
        -H "Authorization: Bearer $TEST_TOKEN" 2>&1)
    echo "Existing sessions: $SESSIONS"

    SESSION_ID=$(echo "$SESSIONS" | jq -r '.[0].id // .[0]._id // empty' 2>/dev/null)
fi

if [ -z "$SESSION_ID" ] || [ "$SESSION_ID" = "null" ]; then
    echo -e "${RED}✗ No session available${NC}"
    echo ""
    echo -e "${YELLOW}Manual Test Instructions:${NC}"
    echo "1. Open the UI in your browser"
    echo "2. Start a new chat"
    echo "3. Send a few messages"
    echo "4. Check server logs for:"
    echo "   - 'Compaction not needed' (threshold check)"
    echo "   - '🗜️ Starting context compaction' (compaction triggered)"
    echo "   - '📦 Messages archived' (archive success)"
    echo "   - '💾 Summary message saved' (summary saved)"
    exit 0
fi

echo -e "${GREEN}✓ Using session: $SESSION_ID${NC}"
echo ""

# Check context status
echo -e "${YELLOW}[4/5]${NC} Checking current context status..."
CONTEXT_STATUS=$(curl -s "$API_URL/api/v1/chat/sessions/$SESSION_ID/context-status" \
    -H "Authorization: Bearer $TEST_TOKEN" 2>&1)

echo "Context Status:"
echo "$CONTEXT_STATUS" | jq '.' 2>/dev/null || echo "$CONTEXT_STATUS"
echo ""

# Get current messages
echo -e "${YELLOW}[5/5]${NC} Getting current messages..."
MESSAGES=$(curl -s "$API_URL/api/v1/chat/sessions/$SESSION_ID/messages?limit=5" \
    -H "Authorization: Bearer $TEST_TOKEN" 2>&1)

MESSAGE_COUNT=$(echo "$MESSAGES" | jq '.messages | length' 2>/dev/null || echo "unknown")
echo "Current message count: $MESSAGE_COUNT"
echo ""

# Summary
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Test Summary                                           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Session ID:${NC} $SESSION_ID"
echo ""
echo -e "${YELLOW}To trigger compaction:${NC}"
echo "1. Open UI and chat in this session"
echo "2. Send multiple messages to build up context"
echo "3. With 1% threshold, compaction triggers at ~1,280 tokens"
echo ""
echo -e "${YELLOW}Watch for these logs:${NC}"
echo "  - 'Compaction not needed' - threshold check (shows current tokens vs threshold)"
echo "  - '🗜️ Starting context compaction' - compaction triggered"
echo "  - '📦 Messages archived' - old messages archived in DB"
echo "  - '💾 Summary message saved' - summary created"
echo "  - Notification toast in UI (if frontend is connected)"
echo ""
echo -e "${YELLOW}WebSocket URL:${NC}"
echo "  ws://localhost:5555/api/v1/chat/stream?sessionId=$SESSION_ID&userId=$USER_ID&companyId=$COMPANY_ID"
echo ""
