package handlers

import (
	"testing"

	"go.uber.org/zap"
)

func TestExtractFilePathFromPatch(t *testing.T) {
	// Create handler with a test logger
	logger := zap.NewNop()
	handler := &FilesystemToolHandler{
		logger: logger,
	}

	tests := []struct {
		name        string
		patch       string
		wantPath    string
		wantErr     bool
		description string
	}{
		{
			name: "Standard unified diff format (--- a/file)",
			patch: `--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 modified
 Line 3`,
			wantPath:    "test.txt",
			wantErr:     false,
			description: "Extract from '--- a/file' header",
		},
		{
			name: "Standard unified diff format (+++ b/file)",
			patch: `+++ b/test.txt
@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 modified
 Line 3`,
			wantPath:    "test.txt",
			wantErr:     false,
			description: "Extract from '+++ b/file' header",
		},
		{
			name: "Simple format with timestamp",
			patch: `--- test.txt	2025-01-01 12:00:00.000000000 +0000
+++ test.txt	2025-01-01 12:00:01.000000000 +0000
@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 modified
 Line 3`,
			wantPath:    "test.txt",
			wantErr:     false,
			description: "Extract with timestamp (split on tab)",
		},
		{
			name: "Custom format (*** Update File:)",
			patch: `*** Begin Patch
*** Update File: test.txt
@@
-Line 2
+Line 2 modified
*** End Patch`,
			wantPath:    "test.txt",
			wantErr:     false,
			description: "Extract from custom '*** Update File:' header",
		},
		{
			name: "Path with directory (--- a/path/to/file.txt)",
			patch: `--- a/src/handlers/test.go
+++ b/src/handlers/test.go
@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 modified
 Line 3`,
			wantPath:    "src/handlers/test.go",
			wantErr:     false,
			description: "Extract nested path",
		},
		{
			name: "No file path in patch",
			patch: `@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 modified
 Line 3`,
			wantPath:    "",
			wantErr:     true,
			description: "Should error when no path headers found",
		},
		{
			name: "/dev/null should be ignored",
			patch: `--- /dev/null
+++ b/test.txt
@@ -0,0 +1,3 @@
+Line 1
+Line 2
+Line 3`,
			wantPath:    "test.txt",
			wantErr:     false,
			description: "Ignore /dev/null and use next header",
		},
		{
			name: "Absolute path (/tmp/test.txt)",
			patch: `--- a/tmp/test.txt
+++ b/tmp/test.txt
@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 modified
 Line 3`,
			wantPath:    "/tmp/test.txt",
			wantErr:     false,
			description: "Extract absolute path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := handler.extractFilePathFromPatch(tt.patch)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractFilePathFromPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotPath != tt.wantPath {
				t.Errorf("extractFilePathFromPatch() = %v, want %v\nDescription: %s", gotPath, tt.wantPath, tt.description)
			}

			if !tt.wantErr {
				t.Logf("✓ %s: extracted '%s'", tt.description, gotPath)
			} else {
				t.Logf("✓ %s: correctly returned error", tt.description)
			}
		})
	}
}
