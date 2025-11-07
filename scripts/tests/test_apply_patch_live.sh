#!/bin/bash

echo "=================================================="
echo "Testing apply_patch Tool with Live MCP Server"
echo "=================================================="
echo

# Create test file
TEST_FILE="/tmp/test_patch_file.txt"
cat > "$TEST_FILE" <<'EOF'
Line 1: Original content
Line 2: Original content
Line 3: Original content
EOF

echo "✓ Created test file: $TEST_FILE"
cat "$TEST_FILE"
echo

# Test 1: Patch WITH explicit filePath (backwards compatibility)
echo "Test 1: With explicit filePath (backwards compatible)"
echo "------------------------------------------------------"

PATCH_1='@@ -1,3 +1,3 @@\n Line 1: Original content\n-Line 2: Original content\n+Line 2: PATCHED with explicit path\n Line 3: Original content'

curl -s -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -d "{
    \"jsonrpc\": \"2.0\",
    \"id\": 1,
    \"method\": \"tools/call\",
    \"params\": {
      \"name\": \"apply_patch\",
      \"arguments\": {
        \"filePath\": \"$TEST_FILE\",
        \"patch\": \"$PATCH_1\"
      }
    }
  }" | jq -r '.result.content[0].text // .error.message' 2>/dev/null || echo "MCP session not initialized"

echo
echo "File contents after Test 1:"
cat "$TEST_FILE"
echo
echo

# Reset file
cat > "$TEST_FILE" <<'EOF'
Line 1: Original content
Line 2: Original content
Line 3: Original content
EOF

# Test 2: Patch WITHOUT explicit filePath (NEW FEATURE - path extraction)
echo "Test 2: WITHOUT explicit filePath (path extracted from patch)"
echo "--------------------------------------------------------------"

PATCH_2="--- a$TEST_FILE\n+++ b$TEST_FILE\n@@ -1,3 +1,3 @@\n Line 1: Original content\n-Line 2: Original content\n+Line 2: PATCHED via path extraction!\n Line 3: Original content"

curl -s -X POST http://localhost:7095/mcp \
  -H "Content-Type: application/json" \
  -d "{
    \"jsonrpc\": \"2.0\",
    \"id\": 2,
    \"method\": \"tools/call\",
    \"params\": {
      \"name\": \"apply_patch\",
      \"arguments\": {
        \"patch\": \"$PATCH_2\"
      }
    }
  }" | jq -r '.result.content[0].text // .error.message' 2>/dev/null || echo "MCP session not initialized"

echo
echo "File contents after Test 2:"
cat "$TEST_FILE"
echo
echo

# Test using the actual Unix patch command to verify patch format
echo "Test 3: Verifying patch format with Unix patch command"
echo "-------------------------------------------------------"

cat > "$TEST_FILE" <<'EOF'
Line 1: Original content
Line 2: Original content
Line 3: Original content
EOF

cat > /tmp/test.patch <<EOF
--- a$TEST_FILE
+++ b$TEST_FILE
@@ -1,3 +1,3 @@
 Line 1: Original content
-Line 2: Original content
+Line 2: PATCHED via Unix patch
 Line 3: Original content
EOF

echo "Patch file contents:"
cat /tmp/test.patch
echo

patch -p0 < /tmp/test.patch 2>&1

echo
echo "File contents after Unix patch:"
cat "$TEST_FILE"
echo

echo "=================================================="
echo "Summary:"
echo "- Test 1: Explicit filePath (backward compatible)"
echo "- Test 2: Path extracted from patch headers (NEW!)"
echo "- Test 3: Unix patch verification"
echo "=================================================="
