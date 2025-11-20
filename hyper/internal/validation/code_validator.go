package validation

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ValidationResult represents the outcome of code validation
type ValidationResult struct {
	Passed       bool              `json:"passed"`
	Errors       []ValidationError `json:"errors"`
	Warnings     []ValidationError `json:"warnings"`
	CheckedFiles []string          `json:"checkedFiles"`
	Duration     time.Duration     `json:"duration"`
	Command      string            `json:"command"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Severity string `json:"severity"` // "error" or "warning"
}

// CodeValidator handles code validation for different file types
type CodeValidator struct {
	logger      *zap.Logger
	projectRoot string
}

// NewCodeValidator creates a new code validator
func NewCodeValidator(logger *zap.Logger, projectRoot string) *CodeValidator {
	return &CodeValidator{
		logger:      logger,
		projectRoot: projectRoot,
	}
}

// ValidateFiles runs appropriate validation based on file extensions
func (v *CodeValidator) ValidateFiles(ctx context.Context, files []string) (*ValidationResult, error) {
	if len(files) == 0 {
		return &ValidationResult{Passed: true}, nil
	}

	// Group files by type
	var tsFiles, goFiles, pyFiles []string
	for _, file := range files {
		ext := filepath.Ext(file)
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			tsFiles = append(tsFiles, file)
		case ".go":
			goFiles = append(goFiles, file)
		case ".py":
			pyFiles = append(pyFiles, file)
		}
	}

	// Run appropriate validators
	var allErrors []ValidationError
	var allWarnings []ValidationError
	var checkedFiles []string
	var totalDuration time.Duration

	if len(tsFiles) > 0 {
		result, err := v.validateTypeScript(ctx, tsFiles)
		if err != nil {
			v.logger.Warn("TypeScript validation failed to run", zap.Error(err))
		} else {
			allErrors = append(allErrors, result.Errors...)
			allWarnings = append(allWarnings, result.Warnings...)
			checkedFiles = append(checkedFiles, result.CheckedFiles...)
			totalDuration += result.Duration
		}
	}

	if len(goFiles) > 0 {
		result, err := v.validateGo(ctx, goFiles)
		if err != nil {
			v.logger.Warn("Go validation failed to run", zap.Error(err))
		} else {
			allErrors = append(allErrors, result.Errors...)
			allWarnings = append(allWarnings, result.Warnings...)
			checkedFiles = append(checkedFiles, result.CheckedFiles...)
			totalDuration += result.Duration
		}
	}

	return &ValidationResult{
		Passed:       len(allErrors) == 0,
		Errors:       allErrors,
		Warnings:     allWarnings,
		CheckedFiles: checkedFiles,
		Duration:     totalDuration,
		Command:      fmt.Sprintf("validated %d files", len(checkedFiles)),
	}, nil
}

// validateTypeScript runs TypeScript compiler checks
func (v *CodeValidator) validateTypeScript(ctx context.Context, files []string) (*ValidationResult, error) {
	start := time.Now()

	// Check if ui directory exists
	uiDir := filepath.Join(v.projectRoot, "ui")

	// Run tsc on ENTIRE project with incremental compilation (respects tsconfig.json)
	// Using --incremental creates .tsbuildinfo cache for 10-15x faster subsequent runs
	// CRITICAL: We must NOT pass specific files to respect tsconfig.json configuration
	// Use --project tsconfig.app.json to ensure we use the app config (with erasableSyntaxOnly)
	args := []string{"tsc", "--project", "tsconfig.app.json", "--noEmit", "--incremental", "--pretty", "false"}

	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = uiDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore error - we parse errors from output
	duration := time.Since(start)

	// Parse ALL TypeScript errors from entire project
	output := stdout.String() + stderr.String()
	allErrors := v.parseTypeScriptOutput(output)

	// Convert modified files to relative paths for filtering
	modifiedFilesSet := make(map[string]bool)
	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			continue
		}
		relPath, err := filepath.Rel(uiDir, absPath)
		if err != nil {
			continue
		}
		// Normalize path separators for comparison
		modifiedFilesSet[filepath.ToSlash(relPath)] = true
	}

	// Filter errors to only include modified files
	var relevantErrors []ValidationError
	for _, err := range allErrors {
		// Normalize error file path for comparison
		normalizedErrFile := filepath.ToSlash(err.File)
		if modifiedFilesSet[normalizedErrFile] {
			relevantErrors = append(relevantErrors, err)
		}
	}

	v.logger.Info("TypeScript validation completed",
		zap.Int("totalErrors", len(allErrors)),
		zap.Int("relevantErrors", len(relevantErrors)),
		zap.Duration("duration", duration),
		zap.Strings("checkedFiles", files))

	return &ValidationResult{
		Passed:       len(relevantErrors) == 0,
		Errors:       relevantErrors,
		CheckedFiles: files,
		Duration:     duration,
		Command:      "npx tsc --project tsconfig.app.json --noEmit --incremental",
	}, nil
}

// validateGo runs go build and go vet
func (v *CodeValidator) validateGo(ctx context.Context, files []string) (*ValidationResult, error) {
	start := time.Now()

	// Get package paths from files
	packages := v.getGoPackages(files)

	var allErrors []ValidationError

	// Run go vet on packages
	for _, pkg := range packages {
		cmd := exec.CommandContext(ctx, "go", "vet", pkg)
		cmd.Dir = v.projectRoot

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			errors := v.parseGoOutput(stderr.String())
			allErrors = append(allErrors, errors...)
		}
	}

	duration := time.Since(start)

	return &ValidationResult{
		Passed:       len(allErrors) == 0,
		Errors:       allErrors,
		CheckedFiles: files,
		Duration:     duration,
		Command:      "go vet",
	}, nil
}

// parseTypeScriptOutput parses tsc output into structured errors
func (v *CodeValidator) parseTypeScriptOutput(output string) []ValidationError {
	var errors []ValidationError
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, " - error TS") || strings.Contains(line, " - warning TS") {
			err := v.parseTypeScriptLine(line)
			if err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

// parseTypeScriptLine parses a single TypeScript error line
// Format: src/file.tsx:10:5 - error TS2304: Cannot find name 'foo'.
func (v *CodeValidator) parseTypeScriptLine(line string) *ValidationError {
	parts := strings.SplitN(line, " - ", 2)
	if len(parts) < 2 {
		return nil
	}

	// Parse file:line:column
	location := parts[0]
	locationParts := strings.Split(location, ":")
	if len(locationParts) < 3 {
		return nil
	}

	file := strings.TrimSpace(locationParts[0])
	lineNum := 0
	colNum := 0
	fmt.Sscanf(locationParts[1], "%d", &lineNum)
	fmt.Sscanf(locationParts[2], "%d", &colNum)

	// Parse error message
	messageParts := strings.SplitN(parts[1], ": ", 2)
	severity := "error"
	code := ""
	message := parts[1]

	if len(messageParts) >= 2 {
		if strings.Contains(messageParts[0], "error") {
			severity = "error"
		} else if strings.Contains(messageParts[0], "warning") {
			severity = "warning"
		}

		// Extract error code (e.g., TS2304)
		codeMatch := strings.Fields(messageParts[0])
		if len(codeMatch) > 0 {
			code = codeMatch[len(codeMatch)-1]
		}

		message = messageParts[1]
	}

	return &ValidationError{
		File:     file,
		Line:     lineNum,
		Column:   colNum,
		Message:  strings.TrimSpace(message),
		Code:     code,
		Severity: severity,
	}
}

// parseGoOutput parses go vet/build output
func (v *CodeValidator) parseGoOutput(output string) []ValidationError {
	var errors []ValidationError
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, ".go:") {
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 3 {
				lineNum := 0
				fmt.Sscanf(parts[1], "%d", &lineNum)

				errors = append(errors, ValidationError{
					File:     parts[0],
					Line:     lineNum,
					Message:  strings.TrimSpace(parts[2]),
					Severity: "error",
				})
			}
		}
	}

	return errors
}

// getGoPackages extracts unique Go packages from file paths
func (v *CodeValidator) getGoPackages(files []string) []string {
	pkgMap := make(map[string]bool)
	for _, file := range files {
		dir := filepath.Dir(file)
		// Convert to package path relative to project root
		relPath, _ := filepath.Rel(v.projectRoot, dir)
		pkgPath := "./" + relPath
		pkgMap[pkgPath] = true
	}

	packages := make([]string, 0, len(pkgMap))
	for pkg := range pkgMap {
		packages = append(packages, pkg)
	}
	return packages
}

// containsPath checks if a file path matches any in the list
func containsPath(file string, paths []string) bool {
	for _, p := range paths {
		if strings.Contains(file, p) || strings.Contains(p, file) {
			return true
		}
	}
	return false
}

// FormatErrorsForAgent formats validation errors for AI agent consumption
func (v *CodeValidator) FormatErrorsForAgent(result *ValidationResult) string {
	if result.Passed {
		return "✅ All validation checks passed - no errors found"
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("❌ Found %d error(s) that must be fixed:\n\n", len(result.Errors)))

	for i, err := range result.Errors {
		buf.WriteString(fmt.Sprintf("%d. %s:%d:%d - %s\n",
			i+1, err.File, err.Line, err.Column, err.Message))
		if err.Code != "" {
			buf.WriteString(fmt.Sprintf("   Error Code: %s\n", err.Code))
		}
	}

	buf.WriteString("\n⚠️  You MUST fix these errors before completing the task.\n")
	buf.WriteString("Read the affected files, understand the errors, and apply fixes.\n")

	return buf.String()
}
