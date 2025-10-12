package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ApplyPatchTool applies unified diff patches to files
type ApplyPatchTool struct{}

// ApplyPatchInput represents the input schema for patch application
type ApplyPatchInput struct {
	FilePath string `json:"filePath"`
	Patch    string `json:"patch"`
	DryRun   bool   `json:"dryRun,omitempty"` // validate without writing
}

// ApplyPatchOutput represents the output from patch application
type ApplyPatchOutput struct {
	Success      bool     `json:"success"`
	LinesChanged int      `json:"linesChanged"`
	Errors       []string `json:"errors,omitempty"`
}

// Hunk represents a patch hunk
type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []string
}

// Name returns the tool name
func (a *ApplyPatchTool) Name() string {
	return "applyPatch"
}

// Description returns the tool description
func (a *ApplyPatchTool) Description() string {
	return "Apply unified diff patches to files. Supports dry-run mode for validation. Parses unified diff format and applies changes line-by-line."
}

// Call applies the patch
func (a *ApplyPatchTool) Call(ctx context.Context, input string) (string, error) {
	var patchInput ApplyPatchInput
	if err := json.Unmarshal([]byte(input), &patchInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate path
	if err := validatePath(patchInput.FilePath); err != nil {
		return "", err
	}

	// Check file exists
	if _, err := os.Stat(patchInput.FilePath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	// Read original file
	content, err := os.ReadFile(patchInput.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Parse patch
	hunks, err := parsePatch(patchInput.Patch)
	if err != nil {
		output := ApplyPatchOutput{
			Success: false,
			Errors:  []string{fmt.Sprintf("patch parse error: %v", err)},
		}
		result, _ := json.Marshal(output)
		return string(result), nil
	}

	// Apply hunks
	var errors []string
	linesChanged := 0
	result := make([]string, 0, len(lines))

	lineIdx := 0
	for _, hunk := range hunks {
		// Copy lines before hunk
		for lineIdx < hunk.OldStart-1 && lineIdx < len(lines) {
			result = append(result, lines[lineIdx])
			lineIdx++
		}

		// Apply hunk
		hunkResult, changed, err := applyHunk(lines, lineIdx, hunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("hunk at line %d: %v", hunk.OldStart, err))
			continue
		}

		result = append(result, hunkResult...)
		lineIdx += hunk.OldLines
		linesChanged += changed
	}

	// Copy remaining lines
	for lineIdx < len(lines) {
		result = append(result, lines[lineIdx])
		lineIdx++
	}

	// Write result if not dry-run
	if !patchInput.DryRun && len(errors) == 0 {
		newContent := strings.Join(result, "\n")
		if err := os.WriteFile(patchInput.FilePath, []byte(newContent), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
	}

	output := ApplyPatchOutput{
		Success:      len(errors) == 0,
		LinesChanged: linesChanged,
		Errors:       errors,
	}

	resultJSON, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(resultJSON), nil
}

// parsePatch parses unified diff format
func parsePatch(patch string) ([]Hunk, error) {
	var hunks []Hunk
	scanner := bufio.NewScanner(strings.NewReader(patch))

	for scanner.Scan() {
		line := scanner.Text()

		// Look for hunk header: @@ -oldStart,oldLines +newStart,newLines @@
		if strings.HasPrefix(line, "@@") {
			parts := strings.Split(line, "@@")
			if len(parts) < 2 {
				continue
			}

			hunkInfo := strings.TrimSpace(parts[1])
			oldNew := strings.Fields(hunkInfo)
			if len(oldNew) < 2 {
				continue
			}

			hunk := Hunk{}

			// Parse old range
			oldRange := strings.TrimPrefix(oldNew[0], "-")
			if strings.Contains(oldRange, ",") {
				oldParts := strings.Split(oldRange, ",")
				hunk.OldStart, _ = strconv.Atoi(oldParts[0])
				hunk.OldLines, _ = strconv.Atoi(oldParts[1])
			} else {
				hunk.OldStart, _ = strconv.Atoi(oldRange)
				hunk.OldLines = 1
			}

			// Parse new range
			newRange := strings.TrimPrefix(oldNew[1], "+")
			if strings.Contains(newRange, ",") {
				newParts := strings.Split(newRange, ",")
				hunk.NewStart, _ = strconv.Atoi(newParts[0])
				hunk.NewLines, _ = strconv.Atoi(newParts[1])
			} else {
				hunk.NewStart, _ = strconv.Atoi(newRange)
				hunk.NewLines = 1
			}

			// Read hunk lines
			hunk.Lines = []string{}
			for scanner.Scan() {
				hunkLine := scanner.Text()
				if strings.HasPrefix(hunkLine, "@@") || strings.HasPrefix(hunkLine, "diff ") || strings.HasPrefix(hunkLine, "---") || strings.HasPrefix(hunkLine, "+++") {
					// Next hunk or file
					break
				}
				hunk.Lines = append(hunk.Lines, hunkLine)
			}

			hunks = append(hunks, hunk)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(hunks) == 0 {
		return nil, fmt.Errorf("no valid hunks found in patch")
	}

	return hunks, nil
}

// applyHunk applies a single hunk to the lines
func applyHunk(lines []string, startIdx int, hunk Hunk) ([]string, int, error) {
	result := []string{}
	changed := 0
	oldIdx := 0

	for _, line := range hunk.Lines {
		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case ' ':
			// Context line - must match
			if startIdx+oldIdx >= len(lines) {
				return nil, 0, fmt.Errorf("hunk extends beyond file")
			}
			if strings.TrimSpace(lines[startIdx+oldIdx]) != strings.TrimSpace(line[1:]) {
				return nil, 0, fmt.Errorf("context mismatch at line %d: expected '%s', got '%s'",
					startIdx+oldIdx+1, strings.TrimSpace(line[1:]), strings.TrimSpace(lines[startIdx+oldIdx]))
			}
			result = append(result, line[1:])
			oldIdx++

		case '-':
			// Delete line - must match what we're deleting
			if startIdx+oldIdx >= len(lines) {
				return nil, 0, fmt.Errorf("delete extends beyond file")
			}
			// Verify the line matches before deleting
			if len(line) > 1 && strings.TrimSpace(lines[startIdx+oldIdx]) != strings.TrimSpace(line[1:]) {
				return nil, 0, fmt.Errorf("delete line mismatch at line %d: expected '%s', got '%s'",
					startIdx+oldIdx+1, strings.TrimSpace(line[1:]), strings.TrimSpace(lines[startIdx+oldIdx]))
			}
			oldIdx++
			changed++

		case '+':
			// Add line
			result = append(result, line[1:])
			changed++

		default:
			// Unknown line type
			continue
		}
	}

	return result, changed, nil
}
