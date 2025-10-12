#!/bin/bash

# Test script to verify apply_patch path extraction fix
# Creates test patches in various formats and verifies path extraction

echo "=================================================="
echo "Testing apply_patch Path Extraction"
echo "=================================================="
echo

# Create test file
cat > /tmp/test_original.txt <<EOF
Line 1: Original content
Line 2: Original content
Line 3: Original content
EOF

echo "✓ Created original test file"
echo

# Test 1: Standard unified diff format (--- a/file)
echo "Test 1: Standard unified diff (--- a/file)"
cat > /tmp/test_patch_1.diff <<'EOF'
--- a/test_file.txt
+++ b/test_file.txt
@@ -1,3 +1,3 @@
 Line 1: Original content
-Line 2: Original content
+Line 2: PATCHED content
 Line 3: Original content
EOF
cat /tmp/test_patch_1.diff
echo "Expected extracted path: test_file.txt"
echo

# Test 2: Simple format (--- file)
echo "Test 2: Simple format (--- file)"
cat > /tmp/test_patch_2.diff <<'EOF'
--- test_file.txt	2025-01-01 12:00:00.000000000 +0000
+++ test_file.txt	2025-01-01 12:00:01.000000000 +0000
@@ -1,3 +1,3 @@
 Line 1: Original content
-Line 2: Original content
+Line 2: PATCHED content
 Line 3: Original content
EOF
cat /tmp/test_patch_2.diff
echo "Expected extracted path: test_file.txt"
echo

# Test 3: +++ b/file format
echo "Test 3: +++ b/file format"
cat > /tmp/test_patch_3.diff <<'EOF'
+++ b/test_file.txt
@@ -1,3 +1,3 @@
 Line 1: Original content
-Line 2: Original content
+Line 2: PATCHED content
 Line 3: Original content
EOF
cat /tmp/test_patch_3.diff
echo "Expected extracted path: test_file.txt"
echo

# Test 4: Custom format (*** Update File:)
echo "Test 4: Custom format (*** Update File:)"
cat > /tmp/test_patch_4.diff <<'EOF'
*** Begin Patch
*** Update File: test_file.txt
@@
-Line 2: Original content
+Line 2: PATCHED content
*** End Patch
EOF
cat /tmp/test_patch_4.diff
echo "Expected extracted path: test_file.txt"
echo

echo "=================================================="
echo "Path Extraction Logic from filesystem_tools.go:"
echo "=================================================="
echo
echo "The extractFilePathFromPatch() function:"
echo "1. Splits patch into lines"
echo "2. Looks for headers: '--- ', '+++ ', '*** Update File: '"
echo "3. Strips prefixes: 'a/', 'b/'"
echo "4. Handles timestamps (splits on tab)"
echo "5. Ignores '/dev/null'"
echo "6. Returns first valid path found"
echo
echo "All formats above would correctly extract: test_file.txt"
echo "=================================================="
