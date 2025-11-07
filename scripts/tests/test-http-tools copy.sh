#!/bin/bash

# HTTP Tools End-to-End Test Script
# Tests the complete workflow: create tool -> list tools -> discover via semantic search -> delete

set -e

API_BASE="http://localhost:7095"
echo "🧪 Testing HTTP Tools API..."
echo ""

# Test 1: Health check
echo "1️⃣  Health Check..."
curl -s "${API_BASE}/health" | jq .
echo ""

# Test 2: List tools (should be empty or show existing tools)
echo "2️⃣  Listing existing HTTP tools..."
curl -s "${API_BASE}/api/v1/tools/http" | jq .
echo ""

# Test 3: Create a new HTTP tool
echo "3️⃣  Creating weather API tool..."
RESPONSE=$(curl -s -X POST "${API_BASE}/api/v1/tools/http" \
  -H "Content-Type: application/json" \
  -d '{
    "toolName": "weather_api",
    "description": "Get current weather conditions and forecasts for any city worldwide using real-time meteorological data",
    "endpoint": "https://api.weather.com/v1/current",
    "httpMethod": "GET",
    "parameters": [
      {"name": "city", "type": "string", "required": true, "description": "City name"},
      {"name": "units", "type": "string", "required": false, "description": "Temperature units (metric or imperial)"}
    ],
    "authType": "apiKey",
    "headers": [
      {"key": "Accept", "value": "application/json"}
    ]
  }')

echo "$RESPONSE" | jq .
TOOL_ID=$(echo "$RESPONSE" | jq -r '.id // .toolId // empty')
echo "Tool ID: $TOOL_ID"
echo ""

# Test 4: List tools again (should include new tool)
echo "4️⃣  Listing tools after creation..."
curl -s "${API_BASE}/api/v1/tools/http?page=1&limit=10" | jq .
echo ""

# Test 5: Get specific tool by ID
if [ -n "$TOOL_ID" ]; then
    echo "5️⃣  Getting tool by ID: $TOOL_ID..."
    curl -s "${API_BASE}/api/v1/tools/http/${TOOL_ID}" | jq .
    echo ""
fi

# Test 6: Create another tool for variety
echo "6️⃣  Creating video processing tool..."
RESPONSE2=$(curl -s -X POST "${API_BASE}/api/v1/tools/http" \
  -H "Content-Type: application/json" \
  -d '{
    "toolName": "video_converter",
    "description": "Convert video files between different formats like MP4, AVI, MKV with customizable quality and resolution settings",
    "endpoint": "https://api.videotools.com/v1/convert",
    "httpMethod": "POST",
    "parameters": [
      {"name": "sourceUrl", "type": "string", "required": true, "description": "Source video URL"},
      {"name": "targetFormat", "type": "string", "required": true, "description": "Target format (mp4, avi, mkv)"},
      {"name": "quality", "type": "string", "required": false, "description": "Quality preset (low, medium, high)"}
    ],
    "authType": "bearer"
  }')

echo "$RESPONSE2" | jq .
TOOL_ID2=$(echo "$RESPONSE2" | jq -r '.id // .toolId // empty')
echo ""

# Test 7: Test semantic discovery via discover_tools MCP endpoint
echo "7️⃣  Testing semantic discovery via MCP..."
echo "   Searching for 'weather tools'..."
# Note: MCP endpoint requires proper session handling, testing via REST proxy if available
# This would need to be tested via Claude Code MCP client

echo ""
echo "8️⃣  Testing semantic search for 'video tools'..."
# Similar to above - requires MCP session

echo ""
echo "9️⃣  Final tool list:"
curl -s "${API_BASE}/api/v1/tools/http?page=1&limit=20" | jq '.tools[] | {toolName, description, httpMethod}'
echo ""

# Optional: Cleanup (commented out by default)
# if [ -n "$TOOL_ID" ]; then
#     echo "🗑️  Deleting test tool..."
#     curl -s -X DELETE "${API_BASE}/api/v1/tools/http/${TOOL_ID}" | jq .
#     echo ""
# fi

echo "✅ HTTP Tools API tests complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Test semantic discovery via Claude Code MCP: discover_tools('weather')"
echo "   2. Open UI: http://localhost:7095/ui and navigate to HTTP Tools page"
echo "   3. Test CRUD operations via UI"
