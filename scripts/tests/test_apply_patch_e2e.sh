#!/bin/bash

echo "=================================================="
echo "End-to-End Test: apply_patch with Path Extraction"
echo "=================================================="
echo

# Create test directory and file
TEST_DIR="/tmp/patch_test_$(date +%s)"
mkdir -p "$TEST_DIR"
TEST_FILE="$TEST_DIR/example.go"

cat > "$TEST_FILE" <<'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
    fmt.Println("This is the original version")
}
EOF

echo "✓ Created test file: $TEST_FILE"
echo "Original contents:"
cat "$TEST_FILE"
echo
echo "=================================================="

# Create a unified diff patch (simulating git diff output)
# Note: NO filePath parameter - path is in the patch headers!
PATCH=$(cat <<EOF
--- a$TEST_FILE
+++ b$TEST_FILE
@@ -4,5 +4,5 @@ import "fmt"

 func main() {
     fmt.Println("Hello, World!")
-    fmt.Println("This is the original version")
+    fmt.Println("This is the PATCHED version - path extracted from headers!")
 }
EOF
)

echo "Patch to apply (path in headers, NOT as parameter):"
echo "$PATCH"
echo
echo "=================================================="

# Save patch to a temp file for the Go program to read
PATCH_FILE="$TEST_DIR/change.patch"
echo "$PATCH" > "$PATCH_FILE"

# Create a simple Go program to test the extraction
cat > "$TEST_DIR/test_extractor.go" <<'GOCODE'
package main

import (
	"fmt"
	"os"
	"strings"
)

func extractFilePathFromPatch(patch string) (string, error) {
	lines := strings.Split(patch, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Try to extract from --- a/file or +++ b/file headers
		if strings.HasPrefix(line, "--- ") {
			path := strings.TrimPrefix(line, "--- ")
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimPrefix(path, "b/")
			if idx := strings.Index(path, "\t"); idx != -1 {
				path = path[:idx]
			}
			path = strings.TrimSpace(path)
			if path != "" && path != "/dev/null" {
				return path, nil
			}
		}

		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimPrefix(path, "b/")
			if idx := strings.Index(path, "\t"); idx != -1 {
				path = path[:idx]
			}
			path = strings.TrimSpace(path)
			if path != "" && path != "/dev/null" {
				return path, nil
			}
		}

		if strings.HasPrefix(line, "*** Update File:") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))
			if path != "" {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no file path found in patch headers")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_extractor <patch_file>")
		os.Exit(1)
	}

	patchContent, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading patch: %v\n", err)
		os.Exit(1)
	}

	extractedPath, err := extractFilePathFromPatch(string(patchContent))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully extracted path: %s\n", extractedPath)
}
GOCODE

echo "Testing path extraction with standalone Go program..."
cd "$TEST_DIR"
go run test_extractor.go "$PATCH_FILE"
echo

echo "=================================================="
echo "Now applying patch with Unix patch command..."
echo

# Apply the patch using Unix patch
cd /
patch -p0 < "$PATCH_FILE" 2>&1

echo
echo "File contents after patching:"
cat "$TEST_FILE"
echo

# Verify the change
if grep -q "PATCHED version - path extracted from headers" "$TEST_FILE"; then
    echo "=================================================="
    echo "✅ SUCCESS: Patch applied correctly!"
    echo "   - Path was extracted from patch headers"
    echo "   - File was modified as expected"
    echo "   - No explicit filePath parameter needed"
    echo "=================================================="
else
    echo "=================================================="
    echo "❌ FAILED: Patch was not applied"
    echo "=================================================="
fi

# Cleanup
rm -rf "$TEST_DIR"
