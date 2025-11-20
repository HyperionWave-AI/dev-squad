package storage

import (
	"testing"
)

func TestIndexedFolder_GetIncludePatterns(t *testing.T) {
	tests := []struct {
		name     string
		folder   *IndexedFolder
		expected []string
	}{
		{
			name:     "Returns nil when empty to signal use all supported extensions",
			folder:   &IndexedFolder{},
			expected: nil,
		},
		{
			name: "Custom patterns",
			folder: &IndexedFolder{
				IncludePatterns: []string{"*.rs", "*.cpp"},
			},
			expected: []string{"*.rs", "*.cpp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.folder.GetIncludePatterns()

			// Handle nil case
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d patterns, got %d", len(tt.expected), len(result))
				return
			}
			for i, pattern := range result {
				if pattern != tt.expected[i] {
					t.Errorf("Pattern %d: expected %s, got %s", i, tt.expected[i], pattern)
				}
			}
		})
	}
}

func TestIndexedFolder_GetExcludePatterns(t *testing.T) {
	tests := []struct {
		name     string
		folder   *IndexedFolder
		expected []string
	}{
		{
			name:     "Default patterns when empty",
			folder:   &IndexedFolder{},
			expected: []string{"node_modules", "dist", "build", ".git", "vendor", ".next", "coverage", "__pycache__"},
		},
		{
			name: "Custom patterns",
			folder: &IndexedFolder{
				ExcludePatterns: []string{"target", "bin"},
			},
			expected: []string{"target", "bin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.folder.GetExcludePatterns()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d patterns, got %d", len(tt.expected), len(result))
				return
			}
			for i, pattern := range result {
				if pattern != tt.expected[i] {
					t.Errorf("Pattern %d: expected %s, got %s", i, tt.expected[i], pattern)
				}
			}
		})
	}
}

func TestIndexedFolder_GetChunkSize(t *testing.T) {
	tests := []struct {
		name     string
		folder   *IndexedFolder
		expected string
	}{
		{
			name:     "Default chunk size when empty",
			folder:   &IndexedFolder{},
			expected: "m",
		},
		{
			name: "Custom chunk size xs",
			folder: &IndexedFolder{
				ChunkSize: "xs",
			},
			expected: "xs",
		},
		{
			name: "Custom chunk size xl",
			folder: &IndexedFolder{
				ChunkSize: "xl",
			},
			expected: "xl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.folder.GetChunkSize()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestChunkSizeToLines(t *testing.T) {
	tests := []struct {
		size     string
		expected int
	}{
		{"xs", 50},
		{"s", 100},
		{"m", 200},
		{"l", 400},
		{"xl", 800},
		{"invalid", 200}, // default
		{"", 200},        // default
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			result := ChunkSizeToLines(tt.size)
			if result != tt.expected {
				t.Errorf("ChunkSizeToLines(%s) = %d, expected %d", tt.size, result, tt.expected)
			}
		})
	}
}
