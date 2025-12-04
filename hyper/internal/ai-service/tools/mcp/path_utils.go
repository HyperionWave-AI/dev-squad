package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hyper/internal/ai-service/tools"

	"go.uber.org/zap"
)

// correctFilePaths attempts to fix invalid file paths using common correction strategies
// Returns (correctedPaths, unfixablePaths, wasIndexingIssue)
func correctFilePaths(paths []string, logger *zap.Logger) ([]string, []string, bool) {
	if len(paths) == 0 {
		return paths, nil, false
	}

	projectRoot := tools.GetProjectRoot()
	logger.Info("🔧 Validating and correcting file paths",
		zap.Int("pathCount", len(paths)),
		zap.String("projectRoot", projectRoot))

	correctedPaths := make([]string, 0, len(paths))
	unfixablePaths := make([]string, 0)
	hadCorrections := false
	indexingIssuePatterns := 0

	for _, path := range paths {
		if path == "" {
			continue
		}

		// FIX #5: Resolve relative paths against project root, not working directory
		// Relative paths like "./ui/src/..." should be resolved from project root
		resolvedPath := path
		if !filepath.IsAbs(path) {
			// Path is relative - resolve against project root
			resolvedPath = filepath.Join(projectRoot, path)
			logger.Debug("Resolved relative path",
				zap.String("original", path),
				zap.String("resolved", resolvedPath))
		}

		// Check if resolved path exists
		if _, err := os.Stat(resolvedPath); err == nil {
			correctedPaths = append(correctedPaths, resolvedPath)
			logger.Debug("✅ Path valid", zap.String("path", resolvedPath))
			continue
		}

		// Path doesn't exist, try to fix it
		logger.Warn("⚠️  Path does not exist, attempting correction",
			zap.String("originalPath", path),
			zap.String("resolvedPath", resolvedPath))

		fixedPath := tryFixPath(resolvedPath, projectRoot, logger)

		if fixedPath != "" {
			// Verify the fixed path exists
			if _, err := os.Stat(fixedPath); err == nil {
				correctedPaths = append(correctedPaths, fixedPath)
				hadCorrections = true
				logger.Info("✅ Path corrected successfully",
					zap.String("original", path),
					zap.String("corrected", fixedPath))

				// Check if this looks like an indexing issue
				if strings.Contains(path, "/hyper/hyper/") || strings.Contains(path, "/hyper/ui/") {
					indexingIssuePatterns++
				}
			} else {
				unfixablePaths = append(unfixablePaths, path)
				logger.Error("❌ Path correction failed - fixed path still doesn't exist",
					zap.String("original", path),
					zap.String("attempted", fixedPath))
			}
		} else {
			unfixablePaths = append(unfixablePaths, path)
			logger.Error("❌ Could not correct path - no valid corrections found",
				zap.String("path", path))
		}
	}

	// Determine if this is an indexing issue
	isIndexingIssue := indexingIssuePatterns > 0
	if isIndexingIssue {
		logger.Warn("🔍 INDEXING ISSUE DETECTED",
			zap.Int("pathsWithIssue", indexingIssuePatterns),
			zap.String("pattern", "Paths contain duplicate /hyper/ or incorrect /hyper/ui/ prefixes"),
			zap.String("recommendation", "Code index may be storing incorrect paths - consider re-indexing"))
	}

	if hadCorrections {
		logger.Info("🔧 Path correction summary",
			zap.Int("totalPaths", len(paths)),
			zap.Int("corrected", len(correctedPaths)),
			zap.Int("unfixable", len(unfixablePaths)),
			zap.Bool("indexingIssue", isIndexingIssue))
	}

	return correctedPaths, unfixablePaths, isIndexingIssue
}

// extractPatternFiles scans contextSummary and todos for file references
// Returns validated file paths that exist and can be used as pattern references
func extractPatternFiles(contextSummary string, todos []string, projectRoot string, logger *zap.Logger) []string {
	// Common file extensions to look for in references
	// Examples: "follow pattern from HTTPToolsPage.tsx", "similar to ChatSessionList.tsx"
	fileExtensions := []string{".tsx", ".ts", ".jsx", ".js", ".go", ".css", ".html", ".py", ".java"}

	candidateFiles := make(map[string]bool)

	// Combine all text to scan
	allText := contextSummary + " " + strings.Join(todos, " ")

	logger.Debug("Scanning for pattern file references",
		zap.String("contextSummary", contextSummary),
		zap.Strings("todos", todos))

	// Extract file names with extensions
	for _, ext := range fileExtensions {
		// Find all occurrences of words ending with the extension
		// Using a simple approach: split by spaces and look for filenames
		words := strings.Fields(allText)
		for _, word := range words {
			// Clean up punctuation
			cleaned := strings.Trim(word, ".,;:()[]{}\"'`")
			if strings.HasSuffix(cleaned, ext) {
				candidateFiles[cleaned] = true
				logger.Debug("Found potential pattern file reference",
					zap.String("file", cleaned))
			}
		}
	}

	if len(candidateFiles) == 0 {
		logger.Debug("No pattern file references found in context/TODOs")
		return nil
	}

	// Now validate and find full paths for these files
	validatedFiles := []string{}
	searchDirs := []string{
		"ui/src/components",
		"ui/src/pages",
		"ui/src/hooks",
		"ui/src",
		"hyper/internal",
		"hyper/internal/ai-service",
		"hyper/internal/mcp",
		"hyper/internal/server",
	}

	for filename := range candidateFiles {
		var foundPath string

		// Strategy 1: Try as absolute path
		if strings.HasPrefix(filename, "/") {
			if _, err := os.Stat(filename); err == nil {
				foundPath = filename
			}
		}

		// Strategy 2: Try relative to project root
		if foundPath == "" {
			candidate := filepath.Join(projectRoot, filename)
			if _, err := os.Stat(candidate); err == nil {
				foundPath = candidate
			}
		}

		// Strategy 3: Search in common directories
		if foundPath == "" {
			for _, dir := range searchDirs {
				candidate := filepath.Join(projectRoot, dir, filename)
				if _, err := os.Stat(candidate); err == nil {
					foundPath = candidate
					break
				}
			}
		}

		// Strategy 4: Try finding with find command (last resort)
		if foundPath == "" {
			logger.Debug("Searching for pattern file using find command",
				zap.String("filename", filename))
			// Use find to locate the file
			// SECURITY: Use exec.Command with separate arguments to prevent command injection
			// DO NOT use bash -c or fmt.Sprintf with user-controlled input
			cmd := exec.Command("find", projectRoot, "-name", filename, "-type", "f")
			output, err := cmd.Output()
			if err == nil && len(output) > 0 {
				// Take first result only (equivalent to | head -1)
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				if len(lines) > 0 && lines[0] != "" {
					foundPath = lines[0]
					if _, err := os.Stat(foundPath); err != nil {
						foundPath = "" // Invalid path
					}
				}
			}
		}

		if foundPath != "" {
			validatedFiles = append(validatedFiles, foundPath)
			logger.Info("✅ Validated pattern reference file",
				zap.String("filename", filename),
				zap.String("fullPath", foundPath))
		} else {
			logger.Warn("⚠️  Pattern reference file not found",
				zap.String("filename", filename),
				zap.String("suggestion", "File mentioned in context but doesn't exist - coordinator may need to verify this reference"))
		}
	}

	return validatedFiles
}

// tryFixPath attempts multiple strategies to correct an invalid file path
func tryFixPath(path string, projectRoot string, logger *zap.Logger) string {
	strategies := []struct {
		name string
		fix  func(string) string
	}{
		{
			name: "Remove duplicate /hyper/hyper/ pattern",
			fix: func(p string) string {
				return strings.Replace(p, "/hyper/hyper/", "/hyper/", 1)
			},
		},
		{
			name: "Replace /hyper/ui/ with /ui/",
			fix: func(p string) string {
				return strings.Replace(p, "/hyper/ui/", "/ui/", 1)
			},
		},
		{
			name: "Remove leading /hyper/ if path starts with it",
			fix: func(p string) string {
				if strings.HasPrefix(p, "/hyper/") {
					return strings.TrimPrefix(p, "/hyper")
				}
				return ""
			},
		},
		{
			name: "Prepend project root to relative path",
			fix: func(p string) string {
				// Remove leading ./ if present
				cleaned := strings.TrimPrefix(p, "./")
				return filepath.Join(projectRoot, cleaned)
			},
		},
		{
			name: "Search in common UI directories",
			fix: func(p string) string {
				basename := filepath.Base(p)
				commonDirs := []string{
					"ui/src/components",
					"ui/src/pages",
					"ui/src",
				}
				for _, dir := range commonDirs {
					candidate := filepath.Join(projectRoot, dir, basename)
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
				return ""
			},
		},
		{
			name: "Search in common backend directories",
			fix: func(p string) string {
				basename := filepath.Base(p)
				commonDirs := []string{
					"hyper/internal/ai-service",
					"hyper/internal/mcp",
					"hyper/internal",
				}
				for _, dir := range commonDirs {
					candidate := filepath.Join(projectRoot, dir, basename)
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
				return ""
			},
		},
	}

	for _, strategy := range strategies {
		fixed := strategy.fix(path)
		if fixed != "" && fixed != path {
			logger.Debug("Trying correction strategy",
				zap.String("strategy", strategy.name),
				zap.String("original", path),
				zap.String("candidate", fixed))

			// Check if this fixed path exists
			if _, err := os.Stat(fixed); err == nil {
				logger.Info("✅ Correction strategy successful",
					zap.String("strategy", strategy.name),
					zap.String("fixed", fixed))
				return fixed
			}
		}
	}

	return "" // No correction worked
}

// normalizePathForComparison cleans and normalizes a path for comparison
// Handles ./, ../, and ensures consistent format for matching
func normalizePathForComparison(path string, projectRoot string) []string {
	// Clean the path to remove ./ and ../ components
	cleaned := filepath.Clean(path)

	// Generate multiple variants for matching
	variants := []string{cleaned}

	// Add absolute version if relative
	if !filepath.IsAbs(cleaned) {
		absPath := filepath.Join(projectRoot, cleaned)
		variants = append(variants, filepath.Clean(absPath))
	}

	// Add relative version if absolute
	if filepath.IsAbs(cleaned) {
		relPath, err := filepath.Rel(projectRoot, cleaned)
		if err == nil {
			variants = append(variants, filepath.Clean(relPath))
		}
	}

	return variants
}
