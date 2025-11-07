#!/bin/bash

# Test script for knowledge voting functionality
# Tests: vote creation, vote retrieval, duplicate vote (upsert), invalid votes, 10-word limit

set -e

BASE_URL="http://localhost:8080/api/v1/knowledge"

echo "=== Knowledge Voting Test Script ==="
echo ""

# Step 1: Create a test knowledge entry
echo "1. Creating test knowledge entry..."
ENTRY_RESPONSE=$(curl -s -X POST "${BASE_URL}/../mcp/tools/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "coordinator_upsert_knowledge",
    "input": {
      "collection": "test-voting",
      "text": "This is a test knowledge entry for voting",
      "metadata": {"test": true}
    }
  }')

ENTRY_ID=$(echo $ENTRY_RESPONSE | jq -r '.result.id')
echo "   Created entry: $ENTRY_ID"
echo ""

# Step 2: Vote with + (upvote)
echo "2. Testing upvote with valid reason..."
VOTE1_RESPONSE=$(curl -s -X POST "${BASE_URL}/entries/${ENTRY_ID}/vote" \
  -H "Content-Type: application/json" \
  -d '{
    "vote": "+",
    "reason": "Very helpful and accurate information"
  }')

echo "   Response: $VOTE1_RESPONSE" | jq '.'
echo ""

# Step 3: Get vote summary
echo "3. Getting vote summary..."
VOTES_RESPONSE=$(curl -s -X GET "${BASE_URL}/entries/${ENTRY_ID}/votes")
echo "   Response: $VOTES_RESPONSE" | jq '.'
echo ""

# Step 4: Update vote to - (downvote) - testing upsert
echo "4. Testing vote update (upsert) - changing to downvote..."
VOTE2_RESPONSE=$(curl -s -X POST "${BASE_URL}/entries/${ENTRY_ID}/vote" \
  -H "Content-Type: application/json" \
  -d '{
    "vote": "-",
    "reason": "Changed my mind not useful"
  }')

echo "   Response: $VOTE2_RESPONSE" | jq '.'
echo ""

# Step 5: Get updated vote summary
echo "5. Getting updated vote summary..."
VOTES_RESPONSE2=$(curl -s -X GET "${BASE_URL}/entries/${ENTRY_ID}/votes")
echo "   Response: $VOTES_RESPONSE2" | jq '.'
echo ""

# Step 6: Test invalid vote value
echo "6. Testing invalid vote value (should fail)..."
INVALID_VOTE=$(curl -s -X POST "${BASE_URL}/entries/${ENTRY_ID}/vote" \
  -H "Content-Type: application/json" \
  -d '{
    "vote": "yes",
    "reason": "This should fail"
  }')

echo "   Response: $INVALID_VOTE" | jq '.'
echo ""

# Step 7: Test 10-word limit violation
echo "7. Testing 10-word limit violation (should fail)..."
LONG_REASON=$(curl -s -X POST "${BASE_URL}/entries/${ENTRY_ID}/vote" \
  -H "Content-Type: application/json" \
  -d '{
    "vote": "+",
    "reason": "This is a very long reason that exceeds the ten word limit maximum"
  }')

echo "   Response: $LONG_REASON" | jq '.'
echo ""

# Step 8: Test exact 10-word reason (should pass)
echo "8. Testing exact 10-word reason (should pass)..."
EXACT_10=$(curl -s -X POST "${BASE_URL}/entries/${ENTRY_ID}/vote" \
  -H "Content-Type: application/json" \
  -d '{
    "vote": "+",
    "reason": "One two three four five six seven eight nine ten"
  }')

echo "   Response: $EXACT_10" | jq '.'
echo ""

# Step 9: Test voting on non-existent entry
echo "9. Testing vote on non-existent entry (should fail)..."
NONEXISTENT=$(curl -s -X POST "${BASE_URL}/entries/00000000-0000-0000-0000-000000000000/vote" \
  -H "Content-Type: application/json" \
  -d '{
    "vote": "+",
    "reason": "Should not work"
  }')

echo "   Response: $NONEXISTENT" | jq '.'
echo ""

# Step 10: Get final vote summary
echo "10. Getting final vote summary..."
FINAL_VOTES=$(curl -s -X GET "${BASE_URL}/entries/${ENTRY_ID}/votes")
echo "   Response: $FINAL_VOTES" | jq '.'
echo ""

echo "=== Test Complete ==="
echo ""
echo "Summary:"
echo "- Entry ID: $ENTRY_ID"
echo "- All tests executed"
echo ""
echo "Expected results:"
echo "  1. Entry created successfully"
echo "  2. Upvote recorded with upvotes=1, downvotes=0, netScore=1"
echo "  3. Vote summary retrieved"
echo "  4. Vote updated to downvote with upvotes=0, downvotes=1, netScore=-1"
echo "  5. Updated summary shows downvote"
echo "  6. Invalid vote value rejected"
echo "  7. 11-word reason rejected"
echo "  8. Exact 10-word reason accepted"
echo "  9. Non-existent entry vote rejected"
echo "  10. Final summary shows latest vote state"
