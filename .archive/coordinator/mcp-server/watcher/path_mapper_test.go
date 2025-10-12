package watcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewPathMapper(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty mappings",
			input:    "",
			expected: 0,
		},
		{
			name:     "single mapping",
			input:    "/host/path:/container/path",
			expected: 1,
		},
		{
			name:     "multiple mappings",
			input:    "/host/path1:/container/path1,/host/path2:/container/path2",
			expected: 2,
		},
		{
			name:     "with spaces",
			input:    "/host/path : /container/path",
			expected: 1,
		},
		{
			name:     "invalid format",
			input:    "/host/path",
			expected: 0,
		},
		{
			name:     "mixed valid and invalid",
			input:    "/host/path1:/container/path1,invalid,/host/path2:/container/path2",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPathMapper(tt.input, logger)
			assert.Equal(t, tt.expected, len(pm.mappings))
		})
	}
}

func TestPathMapper_ToContainerPath(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name         string
		mappings     string
		hostPath     string
		expectedPath string
	}{
		{
			name:         "exact match",
			mappings:     "/Users/max/project:/workspace/mount0",
			hostPath:     "/Users/max/project",
			expectedPath: "/workspace/mount0",
		},
		{
			name:         "subdirectory match",
			mappings:     "/Users/max/project:/workspace/mount0",
			hostPath:     "/Users/max/project/src/main.go",
			expectedPath: "/workspace/mount0/src/main.go",
		},
		{
			name:         "no match returns original",
			mappings:     "/Users/max/project:/workspace/mount0",
			hostPath:     "/Users/other/file.go",
			expectedPath: "/Users/other/file.go",
		},
		{
			name:         "multiple mappings - use longest match",
			mappings:     "/Users/max:/workspace,/Users/max/project:/workspace/mount0",
			hostPath:     "/Users/max/project/src/main.go",
			expectedPath: "/workspace/mount0/src/main.go",
		},
		{
			name:         "no mappings returns original",
			mappings:     "",
			hostPath:     "/any/path/file.go",
			expectedPath: "/any/path/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPathMapper(tt.mappings, logger)
			result := pm.ToContainerPath(tt.hostPath)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}

func TestPathMapper_ToHostPath(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		mappings      string
		containerPath string
		expectedPath  string
	}{
		{
			name:          "exact match",
			mappings:      "/Users/max/project:/workspace/mount0",
			containerPath: "/workspace/mount0",
			expectedPath:  "/Users/max/project",
		},
		{
			name:          "subdirectory match",
			mappings:      "/Users/max/project:/workspace/mount0",
			containerPath: "/workspace/mount0/src/main.go",
			expectedPath:  "/Users/max/project/src/main.go",
		},
		{
			name:          "no match returns original",
			mappings:      "/Users/max/project:/workspace/mount0",
			containerPath: "/other/path/file.go",
			expectedPath:  "/other/path/file.go",
		},
		{
			name:          "multiple mappings - use longest match",
			mappings:      "/Users/max:/workspace,/Users/max/project:/workspace/mount0",
			containerPath: "/workspace/mount0/src/main.go",
			expectedPath:  "/Users/max/project/src/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPathMapper(tt.mappings, logger)
			result := pm.ToHostPath(tt.containerPath)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}

func TestPathMapper_HasMappings(t *testing.T) {
	logger := zap.NewNop()

	t.Run("has mappings", func(t *testing.T) {
		pm := NewPathMapper("/host:/container", logger)
		assert.True(t, pm.HasMappings())
	})

	t.Run("no mappings", func(t *testing.T) {
		pm := NewPathMapper("", logger)
		assert.False(t, pm.HasMappings())
	})
}

func TestPathMapper_GetMappings(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns copy of mappings", func(t *testing.T) {
		pm := NewPathMapper("/host1:/container1,/host2:/container2", logger)
		mappings := pm.GetMappings()

		assert.Equal(t, 2, len(mappings))
		assert.Equal(t, "/container1", mappings["/host1"])
		assert.Equal(t, "/container2", mappings["/host2"])

		// Modify returned map shouldn't affect internal state
		mappings["/host3"] = "/container3"
		assert.Equal(t, 2, len(pm.GetMappings()))
	})
}

func TestPathMapper_ValidateContainerPath(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		mappings      string
		containerPath string
		expected      bool
	}{
		{
			name:          "valid mapped path",
			mappings:      "/Users/max:/workspace/mount0",
			containerPath: "/workspace/mount0/project",
			expected:      true,
		},
		{
			name:          "invalid path not in mappings",
			mappings:      "/Users/max:/workspace/mount0",
			containerPath: "/other/path",
			expected:      false,
		},
		{
			name:          "no mappings - all paths valid",
			mappings:      "",
			containerPath: "/any/path",
			expected:      true,
		},
		{
			name:          "multiple mappings - matches one",
			mappings:      "/Users/max:/workspace/mount0,/Users/other:/workspace/mount1",
			containerPath: "/workspace/mount1/file",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPathMapper(tt.mappings, logger)
			result := pm.ValidateContainerPath(tt.containerPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathMapper_BidirectionalTranslation(t *testing.T) {
	logger := zap.NewNop()

	pm := NewPathMapper("/Users/max/project:/workspace/mount0", logger)

	t.Run("host to container to host", func(t *testing.T) {
		hostPath := "/Users/max/project/src/main.go"
		containerPath := pm.ToContainerPath(hostPath)
		backToHost := pm.ToHostPath(containerPath)

		assert.Equal(t, "/workspace/mount0/src/main.go", containerPath)
		assert.Equal(t, hostPath, backToHost)
	})

	t.Run("container to host to container", func(t *testing.T) {
		containerPath := "/workspace/mount0/src/main.go"
		hostPath := pm.ToHostPath(containerPath)
		backToContainer := pm.ToContainerPath(hostPath)

		assert.Equal(t, "/Users/max/project/src/main.go", hostPath)
		assert.Equal(t, containerPath, backToContainer)
	})
}
