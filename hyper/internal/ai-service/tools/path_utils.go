package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var projectRoot string

// InitProjectRoot detects project root (git root or current working directory)
func InitProjectRoot() error {
	// Try git root first
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if output, err := cmd.Output(); err == nil {
		projectRoot = strings.TrimSpace(string(output))
		return nil
	}
	// Fallback to current working directory
	var err error
	projectRoot, err = os.Getwd()
	return err
}

// GetProjectRoot returns the project root directory
func GetProjectRoot() string {
	if projectRoot == "" {
		InitProjectRoot()
	}
	return projectRoot
}

// MapPath converts absolute paths to project-relative paths
// /test.txt → /project/test.txt (virtual path - doesn't exist)
// /var/folders/test.txt → /var/folders/test.txt (real system path - unchanged)
// ./test.txt → ./test.txt (relative path - unchanged)
func MapPath(path string) string {
	if filepath.IsAbs(path) {
		// Check if this is already a real path that exists on the system
		if _, err := os.Stat(path); err == nil {
			// Path exists on filesystem - leave it unchanged (handles test temp dirs)
			return path
		}

		// Path doesn't exist - walk up the directory tree to find a real ancestor
		// This handles nested paths like /var/folders/tmp/subdir/nested/file.txt
		// where intermediate directories don't exist yet
		testPath := path
		for {
			parentDir := filepath.Dir(testPath)
			if parentDir == testPath || parentDir == string(filepath.Separator) {
				// Reached root without finding existing directory
				break
			}

			if _, err := os.Stat(parentDir); err == nil {
				// Found an existing ancestor directory - this is a real system path
				return path
			}

			testPath = parentDir
		}

		// No existing ancestor found - treat as virtual path and map to project root
		// /test.txt → /project/test.txt
		return filepath.Join(GetProjectRoot(), strings.TrimPrefix(path, string(filepath.Separator)))
	}
	return path
}

// StripProjectRoot converts absolute paths to relative paths for AI display
// /Users/max/project/README.md → ./README.md
// /Users/max/project/ → .
// /Users/max/other/file.txt → /Users/max/other/file.txt (unchanged if not under project root)
func StripProjectRoot(absPath string) string {
	projectRoot := GetProjectRoot()

	// Ensure project root ends with separator for clean prefix matching
	if !strings.HasSuffix(projectRoot, string(filepath.Separator)) {
		projectRoot += string(filepath.Separator)
	}

	// Check if path starts with project root
	if strings.HasPrefix(absPath, projectRoot) {
		// Strip project root prefix
		relPath := strings.TrimPrefix(absPath, projectRoot)

		// If empty (was exactly project root), return "."
		if relPath == "" {
			return "."
		}

		// Prepend "./" for relative path
		return "./" + relPath
	}

	// Path is outside project root - return unchanged
	return absPath
}

// IsSystemPath checks if command contains dangerous system paths
func IsSystemPath(command string) bool {
	systemPaths := []string{"/etc/", "/var/", "/sys/", "/usr/bin/", "/usr/sbin/", "/bin/", "/sbin/", "/proc/", "/dev/"}
	cmdLower := strings.ToLower(command)
	for _, sysPath := range systemPaths {
		if strings.Contains(cmdLower, sysPath) {
			return true
		}
	}
	return false
}
