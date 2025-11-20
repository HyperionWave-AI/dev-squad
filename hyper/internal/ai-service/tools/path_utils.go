package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var projectRoot string

// InitProjectRoot detects project root with the following priority:
// 1. PROJECT_ROOT environment variable (if set)
// 2. Git repository root (git rev-parse --show-toplevel)
// 3. Current working directory (os.Getwd)
func InitProjectRoot() error {
	// Priority 1: Check PROJECT_ROOT environment variable
	if envRoot := os.Getenv("PROJECT_ROOT"); envRoot != "" {
		// Validate that the path exists
		if _, err := os.Stat(envRoot); err == nil {
			projectRoot = envRoot
			return nil
		}
		// Path doesn't exist - log warning and continue with fallbacks
		fmt.Fprintf(os.Stderr, "Warning: PROJECT_ROOT=%s does not exist, falling back to auto-detection\n", envRoot)
	}

	// Priority 2: Try git root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if output, err := cmd.Output(); err == nil {
		projectRoot = strings.TrimSpace(string(output))
		return nil
	}

	// Priority 3: Fallback to current working directory
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
	// Allow safe /dev/ redirects (common Unix pattern)
	safeDevPaths := []string{"/dev/null", "/dev/stdout", "/dev/stderr"}
	cmdLower := strings.ToLower(command)

	// First check if command contains any /dev/ references
	if strings.Contains(cmdLower, "/dev/") {
		// Check if all /dev/ references are safe redirects
		isSafe := false
		for _, safePath := range safeDevPaths {
			if strings.Contains(cmdLower, safePath) {
				isSafe = true
				break
			}
		}
		// If /dev/ is present but not a safe redirect, block it
		if !isSafe {
			// Check if it's ONLY safe paths by removing them and checking if /dev/ still exists
			tempCmd := cmdLower
			for _, safePath := range safeDevPaths {
				tempCmd = strings.ReplaceAll(tempCmd, safePath, "")
			}
			// If /dev/ still exists after removing safe paths, it's dangerous
			if strings.Contains(tempCmd, "/dev/") {
				return true
			}
		}
	}

	// Check other system paths
	systemPaths := []string{"/etc/", "/var/", "/sys/", "/usr/bin/", "/usr/sbin/", "/bin/", "/sbin/", "/proc/"}
	for _, sysPath := range systemPaths {
		if strings.Contains(cmdLower, sysPath) {
			return true
		}
	}
	return false
}
