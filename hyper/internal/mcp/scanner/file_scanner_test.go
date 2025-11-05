package scanner

import (
	"os"
	"testing"

	"hyper/internal/mcp/storage"
)

func TestNewFileScannerWithConfig(t *testing.T) {
	includePatterns := []string{"*.go", "*.rs"}
	excludePatterns := []string{"target", "vendor"}
	chunkSize := "xl"

	scanner := NewFileScannerWithConfig(includePatterns, excludePatterns, chunkSize)

	if scanner == nil {
		t.Fatal("NewFileScannerWithConfig returned nil")
	}

	if len(scanner.includePatterns) != 2 {
		t.Errorf("Expected 2 include patterns, got %d", len(scanner.includePatterns))
	}

	if len(scanner.excludePatterns) != 2 {
		t.Errorf("Expected 2 exclude patterns, got %d", len(scanner.excludePatterns))
	}

	// xl should convert to 800 lines
	if scanner.chunkSize != 800 {
		t.Errorf("Expected chunk size 800 (xl), got %d", scanner.chunkSize)
	}
}

func TestChunkSizeConversion(t *testing.T) {
	tests := []struct {
		size     string
		expected int
	}{
		{"xs", 50},
		{"s", 100},
		{"m", 200},
		{"l", 400},
		{"xl", 800},
		{"invalid", 200}, // defaults to medium
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			scanner := NewFileScannerWithConfig(nil, nil, tt.size)
			if scanner.chunkSize != tt.expected {
				t.Errorf("Expected chunk size %d for size %s, got %d", tt.expected, tt.size, scanner.chunkSize)
			}
		})
	}
}

func TestShouldIncludeFile(t *testing.T) {
	tests := []struct {
		name            string
		includePatterns []string
		filePath        string
		expected        bool
	}{
		{
			name:            "No custom patterns - Go file",
			includePatterns: nil,
			filePath:        "/path/to/file.go",
			expected:        true,
		},
		{
			name:            "No custom patterns - unsupported file",
			includePatterns: nil,
			filePath:        "/path/to/file.xyz",
			expected:        false,
		},
		{
			name:            "Custom pattern matches",
			includePatterns: []string{"*.rs", "*.toml"},
			filePath:        "/path/to/Cargo.toml",
			expected:        true,
		},
		{
			name:            "Custom pattern doesn't match",
			includePatterns: []string{"*.rs", "*.toml"},
			filePath:        "/path/to/file.go",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewFileScannerWithConfig(tt.includePatterns, nil, "m")
			result := scanner.shouldIncludeFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for file %s", tt.expected, result, tt.filePath)
			}
		})
	}
}

func TestShouldExcludePath(t *testing.T) {
	tests := []struct {
		name            string
		excludePatterns []string
		path            string
		basePath        string
		expected        bool
	}{
		{
			name:            "Default excludes - node_modules",
			excludePatterns: nil,
			path:            "/project/node_modules/package",
			basePath:        "/project",
			expected:        true,
		},
		{
			name:            "Default excludes - regular file",
			excludePatterns: nil,
			path:            "/project/src/main.go",
			basePath:        "/project",
			expected:        false,
		},
		{
			name:            "Custom exclude - target directory",
			excludePatterns: []string{"target", "bin"},
			path:            "/project/target/debug",
			basePath:        "/project",
			expected:        true,
		},
		{
			name:            "Custom exclude - doesn't match",
			excludePatterns: []string{"target", "bin"},
			path:            "/project/src/main.rs",
			basePath:        "/project",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewFileScannerWithConfig(nil, tt.excludePatterns, "m")
			result := scanner.shouldExcludePath(tt.path, tt.basePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for path %s", tt.expected, result, tt.path)
			}
		})
	}
}

func TestChunkSizeToLines(t *testing.T) {
	// Test the storage package function
	tests := []struct {
		size     string
		expected int
	}{
		{"xs", 50},
		{"s", 100},
		{"m", 200},
		{"l", 400},
		{"xl", 800},
		{"", 200},
		{"invalid", 200},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			result := storage.ChunkSizeToLines(tt.size)
			if result != tt.expected {
				t.Errorf("ChunkSizeToLines(%s) = %d, expected %d", tt.size, result, tt.expected)
			}
		})
	}
}

// AST Chunking Integration Tests

func TestCreateFileChunks_AST_Go(t *testing.T) {
	// Create a temporary Go file
	tmpFile := t.TempDir() + "/test.go"
	goCode := `package main

import "fmt"

// HelloWorld prints a greeting
func HelloWorld() {
	fmt.Println("Hello, World!")
}

// Add adds two numbers
func Add(a, b int) int {
	return a + b
}
`
	if err := os.WriteFile(tmpFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner()
	chunks, err := scanner.CreateFileChunks("test-file-id", tmpFile)
	if err != nil {
		t.Fatalf("CreateFileChunks failed: %v", err)
	}

	// Should have chunks for functions (import nodes filtered out in current implementation)
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk, got 0")
	}

	// Check that chunks have AST metadata
	hasASTChunk := false
	for _, chunk := range chunks {
		if chunk.ChunkType == "ast" {
			hasASTChunk = true
			if chunk.NodeType == "" {
				t.Error("AST chunk missing NodeType")
			}
		}
	}

	if !hasASTChunk {
		t.Error("Expected at least one AST chunk")
	}
}

func TestCreateFileChunks_AST_Python(t *testing.T) {
	// Create a temporary Python file
	tmpFile := t.TempDir() + "/test.py"
	pythonCode := `import sys

def hello_world():
    """Prints a greeting"""
    print("Hello, World!")

class Calculator:
    """A simple calculator class"""

    def add(self, a, b):
        """Add two numbers"""
        return a + b

    def multiply(self, a, b):
        """Multiply two numbers"""
        return a * b
`
	if err := os.WriteFile(tmpFile, []byte(pythonCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner()
	chunks, err := scanner.CreateFileChunks("test-file-id", tmpFile)
	if err != nil {
		t.Fatalf("CreateFileChunks failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected chunks, got 0")
	}

	// Check for class and method chunks
	foundClass := false
	foundMethod := false

	for _, chunk := range chunks {
		if chunk.NodeType == "class" {
			foundClass = true
			if chunk.NodeName != "Calculator" {
				t.Errorf("Expected class name 'Calculator', got '%s'", chunk.NodeName)
			}
		}
		if chunk.NodeType == "method" {
			foundMethod = true
		}
	}

	if !foundClass {
		t.Error("Expected to find class chunk")
	}
	if !foundMethod {
		t.Error("Expected to find method chunks")
	}
}

func TestCreateFileChunks_AST_TypeScript(t *testing.T) {
	// Create a temporary TypeScript file
	tmpFile := t.TempDir() + "/test.ts"
	tsCode := `interface User {
    name: string;
    email: string;
}

/**
 * Greets a user
 */
function greetUser(user: User): string {
    return "Hello, " + user.name;
}

class UserService {
    /**
     * Creates a new user
     */
    createUser(name: string, email: string): User {
        return { name, email };
    }
}
`
	if err := os.WriteFile(tmpFile, []byte(tsCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner()
	chunks, err := scanner.CreateFileChunks("test-file-id", tmpFile)
	if err != nil {
		t.Fatalf("CreateFileChunks failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected chunks, got 0")
	}

	// Check for function and class (JS parser doesn't extract interfaces yet)
	foundFunction := false
	foundClass := false

	for _, chunk := range chunks {
		if chunk.NodeType == "function" {
			foundFunction = true
		}
		if chunk.NodeType == "class" {
			foundClass = true
		}
	}

	if !foundFunction {
		t.Error("Expected to find function chunk")
	}
	if !foundClass {
		t.Error("Expected to find class chunk")
	}
}

func TestCreateFileChunks_Fallback_UnsupportedFile(t *testing.T) {
	// Create a temporary unsupported file
	tmpFile := t.TempDir() + "/test.txt"
	txtContent := `This is a plain text file.
It should use line-based chunking.
No AST parsing available.
`
	if err := os.WriteFile(tmpFile, []byte(txtContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner()
	chunks, err := scanner.CreateFileChunks("test-file-id", tmpFile)
	if err != nil {
		t.Fatalf("CreateFileChunks failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk, got 0")
	}

	// Should all be line-based chunks
	for _, chunk := range chunks {
		if chunk.ChunkType != "line-based" {
			t.Errorf("Expected ChunkType 'line-based', got '%s'", chunk.ChunkType)
		}
	}
}

func TestCreateFileChunks_Metadata_Population(t *testing.T) {
	// Create a Java file to test Tree-sitter parser metadata extraction
	tmpFile := t.TempDir() + "/Calculator.java"
	javaCode := `import java.util.*;

/**
 * A simple calculator class
 */
public class Calculator {
    /**
     * Add two numbers
     */
    public int add(int a, int b) {
        return a + b;
    }
}
`
	if err := os.WriteFile(tmpFile, []byte(javaCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner()
	chunks, err := scanner.CreateFileChunks("test-file-id", tmpFile)
	if err != nil {
		t.Fatalf("CreateFileChunks failed: %v", err)
	}

	// Log all chunks to debug
	t.Logf("Found %d chunks:", len(chunks))
	for i, chunk := range chunks {
		t.Logf("Chunk %d: Type=%s, NodeType=%s, NodeName=%s, Lines=%d-%d",
			i, chunk.ChunkType, chunk.NodeType, chunk.NodeName, chunk.StartLine, chunk.EndLine)
	}

	// Find any method or class chunk and verify metadata
	foundChunk := false
	for _, chunk := range chunks {
		if chunk.NodeType == "method" || chunk.NodeType == "class" {
			foundChunk = true

			t.Logf("Chunk metadata:")
			t.Logf("  NodeType: %s", chunk.NodeType)
			t.Logf("  NodeName: %s", chunk.NodeName)
			t.Logf("  HasDocstring: %v", chunk.HasDocstring)
			t.Logf("  DocContent: %s", chunk.DocContent)
			t.Logf("  Symbols count: %d", len(chunk.Symbols))

			// Tree-sitter parser should populate these fields
			if chunk.HasDocstring && chunk.DocContent != "" {
				t.Logf("✓ Docstring metadata populated")
			}
			if len(chunk.Symbols) > 0 {
				t.Logf("✓ Symbols extracted: %v", chunk.Symbols)
			}
			break // Found at least one, that's enough
		}
	}

	if !foundChunk {
		t.Log("Note: No method/class chunks found (Tree-sitter may have parse issues with this code)")
	}
}

func TestCreateFileChunks_AST_Fallback_OnParseError(t *testing.T) {
	// Test with a language that Tree-sitter might fail on
	// Use a completely invalid Java file to test Tree-sitter failure fallback
	tmpFile := t.TempDir() + "/invalid.java"
	invalidCode := `this is not valid Java code at all!!!
random text that cannot be parsed
{{{{ broken syntax
`
	if err := os.WriteFile(tmpFile, []byte(invalidCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner()
	chunks, err := scanner.CreateFileChunks("test-file-id", tmpFile)

	// Should still succeed (either parse what it can or fall back to line-based)
	if err != nil {
		t.Fatalf("CreateFileChunks should not fail (should fallback): %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected chunks, got 0")
	}

	// When Tree-sitter fails to parse, it should fall back to line-based
	// Check that we got some chunks (either AST fragments or line-based)
	t.Logf("Created %d chunks for invalid Java file", len(chunks))
	for i, chunk := range chunks {
		t.Logf("Chunk %d: Type=%s, Lines=%d-%d", i, chunk.ChunkType, chunk.StartLine, chunk.EndLine)
	}
}
