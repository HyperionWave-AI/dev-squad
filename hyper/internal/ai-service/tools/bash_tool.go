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
	Timeout int    `json:"timeout,omitempty"` // timeout in seconds, default 30
}

// BashOutput represents the output from bash execution
type BashOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Duration int64  `json:"durationMs"`
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
	return "Execute shell commands and return stdout/stderr. Supports timeout (default 30s). Use for system operations, file checks, script execution."
}

// Call executes the bash command
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

	// Set timeout (default 30s)
	timeout := time.Duration(bashInput.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
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

	output := BashOutput{
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.Stderr = string(exitErr.Stderr)
			output.ExitCode = exitErr.ExitCode()
			output.Stdout = string(stdout)
		} else {
			return "", fmt.Errorf("command execution failed: %w", err)
		}
	} else {
		output.Stdout = string(stdout)
		output.ExitCode = 0
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(result), nil
}
