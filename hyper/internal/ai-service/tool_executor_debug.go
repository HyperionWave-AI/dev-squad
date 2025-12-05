package aiservice

import (
	"fmt"
	"log"
	"os"
	"time"
)

// tool_executor_debug.go - Debug Utilities for Tool Execution
//
// This file contains debug logging utilities for investigating tool execution issues.
// The debug log writes to /tmp/tool_executor_debug.log for detailed tracing.

// debugLogFile is the file handle for debug logging
var debugLogFile *os.File

func init() {
	var err error
	debugLogFile, err = os.OpenFile("/tmp/tool_executor_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open debug log file: %v", err)
	}
}

// debugLog writes a timestamped message to the debug log file.
// This is used for detailed investigation of tool execution flow.
func debugLog(format string, args ...interface{}) {
	if debugLogFile != nil {
		timestamp := time.Now().Format("15:04:05.000")
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(debugLogFile, "[%s] %s\n", timestamp, msg)
		debugLogFile.Sync() // Flush immediately
	}
}
