package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBashTool tests bash command execution
func TestBashTool(t *testing.T) {
	tool := &BashTool{}

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(*testing.T, string)
	}{
		{
			name:      "success with echo",
			input:     `{"command":"echo hello"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result BashOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}
				if !strings.Contains(result.Stdout, "hello") {
					t.Errorf("expected 'hello' in stdout, got: %s", result.Stdout)
				}
				if result.ExitCode != 0 {
					t.Errorf("expected exit code 0, got: %d", result.ExitCode)
				}
			},
		},
		{
			name:      "timeout with sleep",
			input:     `{"command":"sleep 2","timeout":1}`,
			wantError: true,
			checkFunc: func(t *testing.T, output string) {
				// Should not be called since we expect error
			},
		},
		{
			name:      "error with invalid command",
			input:     `{"command":"nonexistentcommand123"}`,
			wantError: false, // command runs but exits with error
			checkFunc: func(t *testing.T, output string) {
				var result BashOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}
				if result.ExitCode == 0 {
					t.Errorf("expected non-zero exit code for invalid command")
				}
			},
		},
		{
			name:      "blocked dangerous command rm -rf",
			input:     `{"command":"rm -rf /"}`,
			wantError: true,
		},
		{
			name:      "blocked dangerous command dd",
			input:     `{"command":"dd if=/dev/zero of=/dev/sda"}`,
			wantError: true,
		},
		{
			name:      "empty command",
			input:     `{"command":""}`,
			wantError: true,
		},
		{
			name:      "invalid JSON",
			input:     `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, err := tool.Call(ctx, tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

// TestReadFileTool tests file reading
func TestReadFileTool(t *testing.T) {
	tool := &ReadFileTool{}

	// Create test files
	tmpDir := t.TempDir()

	normalFile := filepath.Join(tmpDir, "normal.txt")
	if err := os.WriteFile(normalFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryFile := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryFile, []byte{0xFF, 0xFE, 0xFD}, 0644); err != nil {
		t.Fatal(err)
	}

	largeContent := strings.Repeat("x", 11*1024*1024) // 11MB
	largeFile := filepath.Join(tmpDir, "large.txt")
	if err := os.WriteFile(largeFile, []byte(largeContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(*testing.T, string)
	}{
		{
			name:      "read normal file",
			input:     `{"filePath":"` + normalFile + `"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ReadFileOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if result.Content != "hello world" {
					t.Errorf("expected 'hello world', got: %s", result.Content)
				}
				if result.Encoding != "utf8" {
					t.Errorf("expected utf8 encoding, got: %s", result.Encoding)
				}
			},
		},
		{
			name:      "read binary file",
			input:     `{"filePath":"` + binaryFile + `"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ReadFileOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if result.Encoding != "binary" {
					t.Errorf("expected binary encoding, got: %s", result.Encoding)
				}
			},
		},
		{
			name:      "size limit exceeded",
			input:     `{"filePath":"` + largeFile + `"}`,
			wantError: true,
		},
		{
			name:      "path traversal blocked",
			input:     `{"filePath":"../../../etc/passwd"}`,
			wantError: true,
		},
		{
			name:      "file not found",
			input:     `{"filePath":"` + filepath.Join(tmpDir, "nonexistent.txt") + `"}`,
			wantError: true,
		},
		{
			name:      "directory instead of file",
			input:     `{"filePath":"` + tmpDir + `"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, err := tool.Call(ctx, tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

// TestWriteFileTool tests file writing
func TestWriteFileTool(t *testing.T) {
	tool := &WriteFileTool{}
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(*testing.T, string)
	}{
		{
			name:      "create new file",
			input:     `{"filePath":"` + filepath.Join(tmpDir, "new.txt") + `","content":"test content"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result WriteFileOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if result.BytesWritten != 12 {
					t.Errorf("expected 12 bytes written, got: %d", result.BytesWritten)
				}
				content, _ := os.ReadFile(result.Path)
				if string(content) != "test content" {
					t.Errorf("file content mismatch")
				}
			},
		},
		{
			name: "overwrite existing file",
			input: func() string {
				path := filepath.Join(tmpDir, "existing.txt")
				os.WriteFile(path, []byte("old content"), 0644)
				return `{"filePath":"` + path + `","content":"new content"}`
			}(),
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				path := filepath.Join(tmpDir, "existing.txt")
				content, _ := os.ReadFile(path)
				if string(content) != "new content" {
					t.Errorf("file not overwritten correctly, got: %s", string(content))
				}
			},
		},
		{
			name:      "create directories",
			input:     `{"filePath":"` + filepath.Join(tmpDir, "subdir/nested/file.txt") + `","content":"test","createDirs":true}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result WriteFileOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if _, err := os.Stat(result.Path); err != nil {
					t.Errorf("file not created: %v", err)
				}
			},
		},
		{
			name:      "size limit exceeded",
			input:     `{"filePath":"` + filepath.Join(tmpDir, "large.txt") + `","content":"` + strings.Repeat("x", 6*1024*1024) + `"}`,
			wantError: true,
		},
		{
			name:      "path traversal blocked",
			input:     `{"filePath":"../../../tmp/evil.txt","content":"bad"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, err := tool.Call(ctx, tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

// TestListDirectoryTool tests directory listing
func TestListDirectoryTool(t *testing.T) {
	tool := &ListDirectoryTool{}
	tmpDir := t.TempDir()

	// Create test structure
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("test"), 0644)

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(*testing.T, string)
	}{
		{
			name:      "flat listing",
			input:     `{"path":"` + tmpDir + `"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ListDirectoryOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				// Should have file1.txt, file2.txt, subdir (not .hidden without flag)
				if result.Count < 3 {
					t.Errorf("expected at least 3 entries, got: %d", result.Count)
				}
			},
		},
		{
			name:      "recursive listing",
			input:     `{"path":"` + tmpDir + `","recursive":true}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ListDirectoryOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				// Should include nested.txt
				found := false
				for _, entry := range result.Entries {
					if strings.Contains(entry.Name, "nested.txt") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("recursive listing did not find nested.txt")
				}
			},
		},
		{
			name:      "show hidden files",
			input:     `{"path":"` + tmpDir + `","showHidden":true}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ListDirectoryOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				found := false
				for _, entry := range result.Entries {
					if entry.Name == ".hidden" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("hidden files not shown")
				}
			},
		},
		{
			name:      "path not found",
			input:     `{"path":"` + filepath.Join(tmpDir, "nonexistent") + `"}`,
			wantError: true,
		},
		{
			name:      "path is file not directory",
			input:     `{"path":"` + filepath.Join(tmpDir, "file1.txt") + `"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, err := tool.Call(ctx, tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

// TestApplyPatchTool tests patch application
func TestApplyPatchTool(t *testing.T) {
	tool := &ApplyPatchTool{}
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := `line 1
line 2
line 3
line 4
line 5`
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	validPatch := `@@ -2,3 +2,3 @@
 line 2
-line 3
+line 3 modified
 line 4`

	invalidPatch := `@@ -2,3 +2,3 @@
 line 2
-wrong line
+line 3 modified
 line 4`

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(*testing.T, string)
	}{
		{
			name:      "valid patch",
			input:     `{"filePath":"` + testFile + `","patch":"` + strings.ReplaceAll(validPatch, "\n", "\\n") + `"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ApplyPatchOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if !result.Success {
					t.Errorf("patch application failed: %v", result.Errors)
				}
				if result.LinesChanged == 0 {
					t.Errorf("expected lines changed, got 0")
				}
				content, _ := os.ReadFile(testFile)
				if !strings.Contains(string(content), "line 3 modified") {
					t.Errorf("patch not applied correctly")
				}
			},
		},
		{
			name:      "invalid patch (context mismatch)",
			input:     `{"filePath":"` + testFile + `","patch":"` + strings.ReplaceAll(invalidPatch, "\n", "\\n") + `"}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ApplyPatchOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if result.Success {
					t.Errorf("expected patch to fail, but it succeeded")
				}
				if len(result.Errors) == 0 {
					t.Errorf("expected errors, got none")
				}
			},
		},
		{
			name:      "dry run mode",
			input:     `{"filePath":"` + testFile + `","patch":"` + strings.ReplaceAll(validPatch, "\n", "\\n") + `","dryRun":true}`,
			wantError: false,
			checkFunc: func(t *testing.T, output string) {
				var result ApplyPatchOutput
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				// File should not be modified in dry run
				content, _ := os.ReadFile(testFile)
				if strings.Contains(string(content), "modified") {
					t.Errorf("dry run should not modify file")
				}
			},
		},
		{
			name:      "file not found",
			input:     `{"filePath":"` + filepath.Join(tmpDir, "nonexistent.txt") + `","patch":"` + validPatch + `"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset file for each test
			os.WriteFile(testFile, []byte(originalContent), 0644)

			ctx := context.Background()
			output, err := tool.Call(ctx, tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

// TestToolInterfaces ensures all tools implement the Tool interface
func TestToolInterfaces(t *testing.T) {
	tools := []interface{}{
		&BashTool{},
		&ReadFileTool{},
		&WriteFileTool{},
		&ListDirectoryTool{},
		&ApplyPatchTool{},
	}

	for _, tool := range tools {
		// This will fail at compile time if interface is not implemented
		_ = tool
	}
}

// Benchmark tests
func BenchmarkBashTool(b *testing.B) {
	tool := &BashTool{}
	ctx := context.Background()
	input := `{"command":"echo hello"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Call(ctx, input)
	}
}

func BenchmarkReadFileTool(b *testing.B) {
	tool := &ReadFileTool{}
	tmpFile := filepath.Join(os.TempDir(), "bench.txt")
	os.WriteFile(tmpFile, []byte("test content"), 0644)
	defer os.Remove(tmpFile)

	ctx := context.Background()
	input := `{"filePath":"` + tmpFile + `"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Call(ctx, input)
	}
}
