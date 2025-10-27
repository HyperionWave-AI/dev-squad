package scanner

import (
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
