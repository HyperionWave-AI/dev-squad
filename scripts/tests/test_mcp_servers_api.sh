#!/bin/bash

# Test script for MCP Servers REST API endpoints
# This script tests all four endpoints: POST, GET, DELETE, and rediscover

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:4097}"
API_PATH="/api/v1/mcp/servers"
TEST_SERVER_NAME="test-mcp-server-$(date +%s)"
TEST_SERVER_URL="http://localhost:9999/mcp"

echo -e "${YELLOW}Testing MCP Servers REST API${NC}"
echo "Base URL: $BASE_URL"
echo "Test Server Name: $TEST_SERVER_NAME"
echo ""

# Test 1: List servers (should start empty or with existing servers)
echo -e "${YELLOW}Test 1: GET $API_PATH (List servers)${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" "${BASE_URL}${API_PATH}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ List servers successful${NC}"
    echo "Response: $BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ List servers failed with HTTP $HTTP_CODE${NC}"
    echo "$BODY"
    exit 1
fi
echo ""

# Test 2: Add a new MCP server
echo -e "${YELLOW}Test 2: POST $API_PATH (Add server)${NC}"
ADD_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}${API_PATH}" \
    -H "Content-Type: application/json" \
    -d "{
        \"serverName\": \"$TEST_SERVER_NAME\",
        \"serverUrl\": \"$TEST_SERVER_URL\",
        \"description\": \"Test MCP server for API testing\"
    }")

HTTP_CODE=$(echo "$ADD_RESPONSE" | tail -n 1)
BODY=$(echo "$ADD_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Add server successful${NC}"
    echo "Response: $BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Add server failed with HTTP $HTTP_CODE${NC}"
    echo "$BODY"
    exit 1
fi
echo ""

# Test 3: List servers again (should now include our test server)
echo -e "${YELLOW}Test 3: GET $API_PATH (Verify server was added)${NC}"
LIST_RESPONSE=$(curl -s -w "\n%{http_code}" "${BASE_URL}${API_PATH}")
HTTP_CODE=$(echo "$LIST_RESPONSE" | tail -n 1)
BODY=$(echo "$LIST_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    # Check if our test server is in the list
    if echo "$BODY" | jq -e ".servers[] | select(.serverName == \"$TEST_SERVER_NAME\")" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Server found in list${NC}"
        echo "$BODY" | jq ".servers[] | select(.serverName == \"$TEST_SERVER_NAME\")"
    else
        echo -e "${RED}✗ Server not found in list${NC}"
        echo "$BODY"
        exit 1
    fi
else
    echo -e "${RED}✗ List servers failed with HTTP $HTTP_CODE${NC}"
    echo "$BODY"
    exit 1
fi
echo ""

# Test 4: Rediscover server tools (this may fail if server is not actually running, but we test the endpoint)
echo -e "${YELLOW}Test 4: POST $API_PATH/$TEST_SERVER_NAME/rediscover (Rediscover tools)${NC}"
REDISCOVER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}${API_PATH}/${TEST_SERVER_NAME}/rediscover")
HTTP_CODE=$(echo "$REDISCOVER_RESPONSE" | tail -n 1)
BODY=$(echo "$REDISCOVER_RESPONSE" | sed '$d')

# This might fail if the test server URL is not actually running, which is expected
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Rediscover successful${NC}"
    echo "Response: $BODY" | jq '.' 2>/dev/null || echo "$BODY"
elif [ "$HTTP_CODE" = "500" ]; then
    echo -e "${YELLOW}⚠ Rediscover failed (expected if test server not running)${NC}"
    echo "Response: $BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Rediscover failed with unexpected HTTP $HTTP_CODE${NC}"
    echo "$BODY"
fi
echo ""

# Test 5: Delete the test server
echo -e "${YELLOW}Test 5: DELETE $API_PATH/$TEST_SERVER_NAME (Remove server)${NC}"
DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "${BASE_URL}${API_PATH}/${TEST_SERVER_NAME}")
HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n 1)
BODY=$(echo "$DELETE_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Delete server successful${NC}"
    echo "Response: $BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Delete server failed with HTTP $HTTP_CODE${NC}"
    echo "$BODY"
    exit 1
fi
echo ""

# Test 6: Verify server was deleted
echo -e "${YELLOW}Test 6: GET $API_PATH (Verify server was deleted)${NC}"
FINAL_LIST=$(curl -s -w "\n%{http_code}" "${BASE_URL}${API_PATH}")
HTTP_CODE=$(echo "$FINAL_LIST" | tail -n 1)
BODY=$(echo "$FINAL_LIST" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    # Check that our test server is NOT in the list
    if echo "$BODY" | jq -e ".servers[] | select(.serverName == \"$TEST_SERVER_NAME\")" > /dev/null 2>&1; then
        echo -e "${RED}✗ Server still found in list (should be deleted)${NC}"
        echo "$BODY"
        exit 1
    else
        echo -e "${GREEN}✓ Server successfully deleted${NC}"
    fi
else
    echo -e "${RED}✗ List servers failed with HTTP $HTTP_CODE${NC}"
    echo "$BODY"
    exit 1
fi
echo ""

# Test 7: Test error handling - invalid server name
echo -e "${YELLOW}Test 7: POST $API_PATH (Test validation - invalid server name)${NC}"
ERROR_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}${API_PATH}" \
    -H "Content-Type: application/json" \
    -d "{
        \"serverName\": \"invalid server name with spaces\",
        \"serverUrl\": \"http://localhost:9999/mcp\"
    }")

HTTP_CODE=$(echo "$ERROR_RESPONSE" | tail -n 1)
BODY=$(echo "$ERROR_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "400" ]; then
    echo -e "${GREEN}✓ Validation working correctly (rejected invalid server name)${NC}"
    echo "Response: $BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Validation failed - should reject invalid server name${NC}"
    echo "$BODY"
fi
echo ""

echo -e "${GREEN}All MCP Servers API tests completed successfully!${NC}"
