#!/bin/bash
# Script to update system prompt via API
# Requires JWT_SECRET environment variable to be set

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Hyperion System Prompt Update Script${NC}"
echo "========================================"
echo

# Check if server is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
  echo -e "${RED}ERROR: Hyperion server is not running on localhost:8080${NC}"
  echo "Please start the server first with: cd hyper && make dev"
  exit 1
fi

# Read the new prompt content
PROMPT_CONTENT=$(cat updated_system_prompt.md)

# Generate a simple JWT for testing (requires JWT_SECRET)
# In production, use proper authentication
if [ -z "$JWT_SECRET" ]; then
  echo -e "${YELLOW}WARNING: JWT_SECRET not set. Using default test secret.${NC}"
  export JWT_SECRET="your-secret-key-here"
fi

# Create JWT payload
# For testing, we'll use a simple approach - in production use proper JWT generation
echo -e "${YELLOW}Note: This script requires proper JWT authentication.${NC}"
echo -e "${YELLOW}You may need to get a valid token from the UI or use proper auth.${NC}"
echo

# Option 1: Use MongoDB directly (recommended)
echo -e "${GREEN}Recommended approach: Use MongoDB script${NC}"
echo "Run: mongosh hyperion_db < update_system_prompt.js"
echo

# Option 2: Manual API call (requires valid JWT)
echo -e "${YELLOW}Alternative: Manual API call${NC}"
echo "1. Get a valid JWT token by logging into the UI"
echo "2. Use browser DevTools to copy the Authorization header"
echo "3. Run:"
echo
cat << 'EOF'
curl -X POST http://localhost:8080/api/v1/ai-settings/system-prompt/versions \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d @- << JSON
{
  "prompt": "$(cat updated_system_prompt.md | jq -Rs .)",
  "description": "Added MCP tool discovery capabilities",
  "activate": true
}
JSON
EOF

echo
echo -e "${GREEN}Files created:${NC}"
echo "  - updated_system_prompt.md (new prompt content)"
echo "  - update_system_prompt.js (MongoDB script)"
echo "  - update_prompt_via_api.sh (this script)"
