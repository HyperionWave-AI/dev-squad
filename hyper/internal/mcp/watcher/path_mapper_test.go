package watcher

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestPathMapperBasic(t *testing.T) {
	logger := zap.NewNop()

	// No mappings
	pm := NewPathMapper("", logger)
	if pm.HasMappings() {
		t.Fatalf("expected no mappings")
	}
	if got := pm.ToContainerPath("/some/path"); got != "/some/path" {
		t.Fatalf("expected unchanged path, got %s", got)
	}
	if !pm.ValidateContainerPath("/any/path") {
		t.Fatalf("validate should be true when no mappings")
	}
	if m := pm.GetMappings(); len(m) != 0 {
		t.Fatalf("expected empty mappings, got %v", m)
	}
}

func TestPathMapperParsingAndTranslation(t *testing.T) {
	logger := zap.NewNop()
	env := "/host:/container,/host/long:/container/long"
	pm := NewPathMapper(env, logger)

	if !pm.HasMappings() {
		t.Fatalf("expected mappings to be present")
	}
	// Ensure both mappings are stored
	expected := map[string]string{"/host": "/container", "/host/long": "/container/long"}
	if !reflect.DeepEqual(pm.GetMappings(), expected) {
		t.Fatalf("mappings mismatch: got %v, want %v", pm.GetMappings(), expected)
	}

	// Longest prefix should be used for translation
	hostPath := "/host/long/sub/file.txt"
	container := pm.ToContainerPath(hostPath)
	if container != "/container/long/sub/file.txt" {
		t.Fatalf("longest prefix translation failed, got %s", container)
	}

	// Shorter prefix translation for non‑long path
	hostPath2 := "/host/other/file.txt"
	container2 := pm.ToContainerPath(hostPath2)
	if container2 != "/container/other/file.txt" {
		t.Fatalf("short prefix translation failed, got %s", container2)
	}

	// Reverse translation
	if host := pm.ToHostPath(container); host != hostPath {
		t.Fatalf("reverse translation failed, got %s", host)
	}
	if host := pm.ToHostPath(container2); host != hostPath2 {
		t.Fatalf("reverse translation failed for short path, got %s", host)
	}
}

func TestPathMapperInvalidEntries(t *testing.T) {
	logger := zap.NewNop()
	// Include an invalid entry and an empty mapping
	env := "badpair,/valid:/mapped,,/also/bad:"
	pm := NewPathMapper(env, logger)
	// Only the valid mapping should be kept
	if len(pm.GetMappings()) != 1 {
		t.Fatalf("expected only one valid mapping, got %d", len(pm.GetMappings()))
	}
	if _, ok := pm.GetMappings()["/valid"]; !ok {
		t.Fatalf("valid mapping missing")
	}
}

func TestValidateContainerPath(t *testing.T) {
	logger := zap.NewNop()
	env := "/host:/container"
	pm := NewPathMapper(env, logger)
	if !pm.ValidateContainerPath("/container/sub/file.go") {
		t.Fatalf("expected valid container path")
	}
	if pm.ValidateContainerPath("/other/path") {
		t.Fatalf("expected invalid container path")
	}
}

// Comprehensive table-driven tests for edge cases and better coverage
func TestPathMapperEdgeCases(t *testing.T) {
	tests := []struct {
		name                    string
		env                     string
		hostPath                string
		expectedContainerPath   string
		containerPath           string
		expectedHostPath        string
		validateContainerPath   string
		expectedValidation    bool
	}{
		{
			name:                  "empty mappings",
			env:                   "",
			hostPath:              "/any/path",
			expectedContainerPath: "/any/path",
			containerPath:         "/any/path",
			expectedHostPath:    "/any/path",
			validateContainerPath: "/any/path",
			expectedValidation:    true,
		},
		{
			name:                  "single mapping with exact match",
			env:                   "/app:/workspace",
			hostPath:              "/app",
			expectedContainerPath: "/workspace",
			containerPath:         "/workspace",
			expectedHostPath:    "/app",
			validateContainerPath: "/workspace",
			expectedValidation:    true,
		},
		{
			name:                  "single mapping with subdirectory",
			env:                   "/app:/workspace",
			hostPath:              "/app/src/main.go",
			expectedContainerPath: "/workspace/src/main.go",
			containerPath:         "/workspace/src/main.go",
			expectedHostPath:    "/app/src/main.go",
			validateContainerPath: "/workspace/src/main.go",
			expectedValidation:    true,
		},
		{
			name:                  "overlapping mappings - longest prefix wins",
			env:                   "/app:/workspace,/app/src:/workspace/src",
			hostPath:              "/app/src/main.go",
			expectedContainerPath: "/workspace/src/main.go",
			containerPath:         "/workspace/src/main.go",
			expectedHostPath:    "/app/src/main.go",
			validateContainerPath: "/workspace/src/main.go",
			expectedValidation:    true,
		},
		{
			name:                  "no matching mapping returns original path",
			env:                   "/app:/workspace",
			hostPath:              "/other/path",
			expectedContainerPath: "/other/path",
			containerPath:         "/other/path",
			expectedHostPath:    "/other/path",
			validateContainerPath: "/other/path",
			expectedValidation:    false,
		},
		{
			name:                  "whitespace handling in mappings",
			env:                   " /app : /workspace , /src:/dst ",
			hostPath:              "/app/file.txt",
			expectedContainerPath: "/workspace/file.txt",
			containerPath:         "/workspace/file.txt",
			expectedHostPath:    "/app/file.txt",
			validateContainerPath: "/workspace/file.txt",
			expectedValidation:    true,
		},
		{
			name:                  "multiple overlapping mappings - longest prefix",
			env:                   "/home/user/project:/workspace,/home/user:/data",
			hostPath:              "/home/user/project/src/main.go",
			expectedContainerPath: "/workspace/src/main.go",
			containerPath:         "/workspace/src/main.go",
			expectedHostPath:    "/home/user/project/src/main.go",
			validateContainerPath: "/workspace/src/main.go",
			expectedValidation:    true,
		},
		{
			name:                  "complex path with special characters",
			env:                   "/home/user/my-app:/workspace/app",
			hostPath:              "/home/user/my-app/src/components/Button.tsx",
			expectedContainerPath: "/workspace/app/src/components/Button.tsx",
			containerPath:         "/workspace/app/src/components/Button.tsx",
			expectedHostPath:    "/home/user/my-app/src/components/Button.tsx",
			validateContainerPath: "/workspace/app/src/components/Button.tsx",
			expectedValidation:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			pm := NewPathMapper(tt.env, logger)

			// Test ToContainerPath
			result := pm.ToContainerPath(tt.hostPath)
			if result != tt.expectedContainerPath {
				t.Errorf("ToContainerPath(%q) = %q, want %q", tt.hostPath, result, tt.expectedContainerPath)
			}

			// Test ToHostPath
			result = pm.ToHostPath(tt.containerPath)
			if result != tt.expectedHostPath {
				t.Errorf("ToHostPath(%q) = %q, want %q", tt.containerPath, result, tt.expectedHostPath)
			}

			// Test ValidateContainerPath
			resultValidation := pm.ValidateContainerPath(tt.validateContainerPath)
			if resultValidation != tt.expectedValidation {
				t.Errorf("ValidateContainerPath(%q) = %v, want %v", tt.validateContainerPath, resultValidation, tt.expectedValidation)
			}

			// Test HasMappings
			expectedHasMappings := tt.env != ""
			if pm.HasMappings() != expectedHasMappings {
				t.Errorf("HasMappings() = %v, want %v", pm.HasMappings(), expectedHasMappings)
			}
		})
	}
}

// Test for invalid mapping formats
func TestPathMapperInvalidFormats(t *testing.T) {
	tests := []struct {
		name          string
		env           string
		expectedLogs  int
	}{
		{
			name:          "missing colon separator",
			env:           "/app/workspace",
			expectedLogs:  0, // Should be skipped, no mappings
		},
		{
			name:          "empty host path",
			env:           ":/workspace",
			expectedLogs:  0, // Should be skipped
		},
		{
			name:          "empty container path",
			env:           "/app:",
			expectedLogs:  0, // Should be skipped
		},
		{
			name:          "multiple colons",
			env:           "/app:/workspace:extra",
			expectedLogs:  0, // Should be skipped
		},
		{
			name:          "valid mixed with invalid",
			env:           "/app:/workspace,badpair,/src:/dst",
			expectedLogs:  2, // Should have 2 valid mappings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a logger that captures log entries
			var logEntries []zapcore.Entry
			logger := zap.New(zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(&testWriter{entries: &logEntries}),
				zapcore.DebugLevel,
			))

			pm := NewPathMapper(tt.env, logger)
			
			if len(pm.GetMappings()) != tt.expectedLogs {
				t.Errorf("Expected %d mappings, got %d", tt.expectedLogs, len(pm.GetMappings()))
			}
		})
	}
}

// Test GetMappings returns a copy
func TestGetMappingsReturnsCopy(t *testing.T) {
	logger := zap.NewNop()
	env := "/app:/workspace"
	pm := NewPathMapper(env, logger)

	mappings := pm.GetMappings()
	originalLen := len(mappings)

	// Modify the returned map
	mappings["/extra"] = "/extra/mapped"

	// Get mappings again - should not include our modification
	newMappings := pm.GetMappings()
	if len(newMappings) != originalLen {
		t.Errorf("GetMappings() should return a copy, but original was modified")
	}
}

// Test edge case: mapping with same prefix but different lengths
func TestPathMapperPrefixEdgeCases(t *testing.T) {
	logger := zap.NewNop()
	
	// Test case where shorter prefix comes first but longer one should win
	env := "/app:/workspace,/application:/workspace/app"
	pm := NewPathMapper(env, logger)

	// Should use the longer prefix match
	result := pm.ToContainerPath("/application/src/main.go")
	expected := "/workspace/app/src/main.go"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test reverse - should also use longest match
	result = pm.ToHostPath("/workspace/app/src/main.go")
	expected = "/application/src/main.go"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Helper type to capture log entries for testing
type testWriter struct {
	entries *[]zapcore.Entry
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Benchmark tests for performance validation
func BenchmarkPathMapperToContainerPath(b *testing.B) {
	logger := zap.NewNop()
	env := "/home/user/project:/workspace,/home/user:/data,/app:/container"
	pm := NewPathMapper(env, logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.ToContainerPath("/home/user/project/src/main.go")
	}
}

func BenchmarkPathMapperToHostPath(b *testing.B) {
	logger := zap.NewNop()
	env := "/home/user/project:/workspace,/home/user:/data,/app:/container"
	pm := NewPathMapper(env, logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.ToHostPath("/workspace/src/main.go")
	}
}

func BenchmarkPathMapperValidateContainerPath(b *testing.B) {
	logger := zap.NewNop()
	env := "/home/user/project:/workspace,/home/user:/data,/app:/container"
	pm := NewPathMapper(env, logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.ValidateContainerPath("/workspace/src/main.go")
	}
}