#!/bin/bash
set -e

echo "🔧 Testing Collection Count Fix"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

API_URL="http://localhost:4097"

echo "Step 1: Check current collection counts"
echo "----------------------------------------"
echo -e "${YELLOW}GET $API_URL/api/v1/knowledge/collections${NC}"
curl -s "$API_URL/api/v1/knowledge/collections" | jq -r '.collections[] | "\(.name): \(.count) entries"'
echo ""

echo "Step 2: Rebuild collection counts"
echo "-----------------------------------"
echo -e "${YELLOW}POST $API_URL/api/v1/knowledge/collections/rebuild-counts${NC}"
REBUILD_RESULT=$(curl -s -X POST "$API_URL/api/v1/knowledge/collections/rebuild-counts")
echo "$REBUILD_RESULT" | jq '.'
echo ""

UPDATED_COUNT=$(echo "$REBUILD_RESULT" | jq -r '.collectionsUpdated')
TOTAL_ENTRIES=$(echo "$REBUILD_RESULT" | jq -r '.totalEntries')

echo -e "${GREEN}✅ Rebuild completed:${NC}"
echo "   - Collections updated: $UPDATED_COUNT"
echo "   - Total entries: $TOTAL_ENTRIES"
echo ""

echo "Step 3: Verify updated counts"
echo "------------------------------"
echo -e "${YELLOW}GET $API_URL/api/v1/knowledge/collections${NC}"
curl -s "$API_URL/api/v1/knowledge/collections" | jq -r '.collections[] | "\(.name): \(.count) entries"'
echo ""

echo "Step 4: Show detailed rebuild stats"
echo "------------------------------------"
echo "$REBUILD_RESULT" | jq -r '.details[] | "  \(.name): \(.oldCount) → \(.actualCount) (\(if .updated then "UPDATED" else "unchanged" end))"'
echo ""

echo -e "${GREEN}✅ Test completed successfully!${NC}"
echo ""
echo "Next steps:"
echo "1. Check UI at http://localhost:4588/ui/knowledge"
echo "2. Verify badges show correct numbers"
echo "3. Add a new knowledge entry and verify count increments"
echo "4. Delete an entry and verify count decrements"
