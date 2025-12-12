package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BashTool executes shell commands with security validations
type BashTool struct{}

// BashInput represents the input schema for bash execution
type BashInput struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // timeout in seconds, default 3600 (60 minutes)
}

var dangerousCommands = []string{
	"rm -rf /",
	"rm -rf /*",
	"dd if=/dev/zero",
	"mkfs",
	"> /dev/sda",
	":(){ :|:& };:",
}

// Name returns the tool name
func (b *BashTool) Name() string {
	return "bash"
}

// Description returns the tool description
func (b *BashTool) Description() string {
	return "Execute shell commands and return stdout/stderr. Supports timeout (default 3600s/60min). Use for system operations, file checks, script execution."
}

// Call executes the bash command and returns raw stdout
func (b *BashTool) Call(ctx context.Context, input string) (string, error) {
	var bashInput BashInput
	if err := json.Unmarshal([]byte(input), &bashInput); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Validate command
	if strings.TrimSpace(bashInput.Command) == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// Check for dangerous commands
	lowerCmd := strings.ToLower(bashInput.Command)
	for _, dangerous := range dangerousCommands {
		if strings.Contains(lowerCmd, dangerous) {
			return "", fmt.Errorf("dangerous command blocked: %s", dangerous)
		}
	}

	// Block system paths
	if IsSystemPath(bashInput.Command) {
		return "", fmt.Errorf("access to system directories is blocked for security")
	}

	// Set timeout (default 3600s / 60 minutes)
	timeout := time.Duration(bashInput.Timeout) * time.Second
	if timeout == 0 {
		timeout = 3600 * time.Second
	}

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", bashInput.Command)
	cmd.Dir = GetProjectRoot() // Set working directory to project root

	stdout, err := cmd.Output()
	duration := time.Since(start).Milliseconds()

	// Check if context was cancelled (timeout)
	if ctx.Err() != nil || cmdCtx.Err() != nil {
		return "", fmt.Errorf("command timed out after %dms", duration)
	}

	// Handle execution errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			// Return stderr if available, otherwise return stdout with error
			if stderr != "" {
				return "", fmt.Errorf("command failed (exit code %d, %dms): %s", exitErr.ExitCode(), duration, stderr)
			}
			// If no stderr, still return the error but with stdout if available
			if len(stdout) > 0 {
				return "", fmt.Errorf("command failed (exit code %d, %dms): %s", exitErr.ExitCode(), duration, string(stdout))
			}
			return "", fmt.Errorf("command failed with exit code %d (%dms)", exitErr.ExitCode(), duration)
		}
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	// Return raw stdout on success
	return string(stdout), nil
}
