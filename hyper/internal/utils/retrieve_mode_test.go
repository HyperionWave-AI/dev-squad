package utils

import "testing"

func TestParseRetrieveMode(t *testing.T) {
	tests := []struct {
		name               string
		mode               string
		expectedType       string
		expectedChunkLines int
		expectedTshirtSize string
	}{
		{
			name:               "chunk-s",
			mode:               "chunk-s",
			expectedType:       "chunk",
			expectedChunkLines: 50,
			expectedTshirtSize: "s",
		},
		{
			name:               "chunk-m",
			mode:               "chunk-m",
			expectedType:       "chunk",
			expectedChunkLines: 100,
			expectedTshirtSize: "m",
		},
		{
			name:               "chunk-l",
			mode:               "chunk-l",
			expectedType:       "chunk",
			expectedChunkLines: 200,
			expectedTshirtSize: "l",
		},
		{
			name:               "chunk-xl",
			mode:               "chunk-xl",
			expectedType:       "chunk",
			expectedChunkLines: 400,
			expectedTshirtSize: "xl",
		},
		{
			name:               "chunk defaults to chunk-l",
			mode:               "chunk",
			expectedType:       "chunk",
			expectedChunkLines: 200,
			expectedTshirtSize: "l",
		},
		{
			name:               "full",
			mode:               "full",
			expectedType:       "full",
			expectedChunkLines: 0,
			expectedTshirtSize: "",
		},
		{
			name:               "unknown defaults to chunk-l",
			mode:               "unknown",
			expectedType:       "chunk",
			expectedChunkLines: 200,
			expectedTshirtSize: "l",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieveType, chunkLines, tshirtSize := ParseRetrieveMode(tt.mode)

			if retrieveType != tt.expectedType {
				t.Errorf("ParseRetrieveMode(%q) retrieveType = %v, want %v", tt.mode, retrieveType, tt.expectedType)
			}
			if chunkLines != tt.expectedChunkLines {
				t.Errorf("ParseRetrieveMode(%q) chunkLines = %v, want %v", tt.mode, chunkLines, tt.expectedChunkLines)
			}
			if tshirtSize != tt.expectedTshirtSize {
				t.Errorf("ParseRetrieveMode(%q) tshirtSize = %v, want %v", tt.mode, tshirtSize, tt.expectedTshirtSize)
			}
		})
	}
}

func TestIsValidRetrieveMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected bool
	}{
		{"chunk-s is valid", "chunk-s", true},
		{"chunk-m is valid", "chunk-m", true},
		{"chunk-l is valid", "chunk-l", true},
		{"chunk-xl is valid", "chunk-xl", true},
		{"chunk is valid", "chunk", true},
		{"full is valid", "full", true},
		{"invalid mode", "invalid", false},
		{"empty string", "", false},
		{"chunk-xs is invalid", "chunk-xs", false},
		{"CHUNK is invalid (case sensitive)", "CHUNK", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidRetrieveMode(tt.mode)
			if result != tt.expected {
				t.Errorf("IsValidRetrieveMode(%q) = %v, want %v", tt.mode, result, tt.expected)
			}
		})
	}
}
