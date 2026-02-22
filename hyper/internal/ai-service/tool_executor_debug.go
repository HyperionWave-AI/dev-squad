package aiservice

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// tool_executor_debug.go - Debug Utilities for Tool Execution
//
// This file contains debug logging utilities for investigating tool execution issues.
// Debug logging is disabled by default and can be enabled with:
// TOOL_EXECUTOR_DEBUG_LOG=true
//
// Optional:
// TOOL_EXECUTOR_DEBUG_LOG_PATH=/path/to/debug.log (default: /tmp/tool_executor_debug.log)

// debugLogFile is the file handle for debug logging
var debugLogFile *os.File

func init() {
	initToolExecutorDebugLog()
}

func initToolExecutorDebugLog() {
	if !isToolExecutorDebugEnabled() {
		return
	}

	logPath := "/tmp/tool_executor_debug.log"
	if customPath := strings.TrimSpace(os.Getenv("TOOL_EXECUTOR_DEBUG_LOG_PATH")); customPath != "" {
		logPath = customPath
	}

	var err error
	debugLogFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open tool executor debug log file (%s): %v", logPath, err)
	}
}

// debugLog writes a timestamped message to the debug log file.
// This is used for detailed investigation of tool execution flow.
func debugLog(format string, args ...interface{}) {
	if debugLogFile != nil {
		timestamp := time.Now().Format("15:04:05.000")
		msg := fmt.Sprintf(format, args...)
		if _, err := fmt.Fprintf(debugLogFile, "[%s] %s\n", timestamp, msg); err != nil {
			log.Printf("Failed to write tool executor debug log: %v", err)
		}
	}
}

func isToolExecutorDebugEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("TOOL_EXECUTOR_DEBUG_LOG")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
