package aiservice

import (
	"os"
	"strings"
	"testing"
)

func closeDebugLogFileForTest(t *testing.T) {
	t.Helper()
	if debugLogFile != nil {
		_ = debugLogFile.Close()
		debugLogFile = nil
	}
}

func TestIsToolExecutorDebugEnabled(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "true", value: "true", expected: true},
		{name: "numeric", value: "1", expected: true},
		{name: "yes with spaces", value: " YES ", expected: true},
		{name: "on mixed case", value: "On", expected: true},
		{name: "false", value: "false", expected: false},
		{name: "empty", value: "", expected: false},
		{name: "random", value: "debug", expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TOOL_EXECUTOR_DEBUG_LOG", tc.value)
			if got := isToolExecutorDebugEnabled(); got != tc.expected {
				t.Fatalf("expected %v for %q, got %v", tc.expected, tc.value, got)
			}
		})
	}
}

func TestDebugLog_WritesToConfiguredFile(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "tool-exec-debug-*.log")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer tmp.Close()

	prev := debugLogFile
	debugLogFile = tmp
	defer func() { debugLogFile = prev }()

	debugLog("tool=%s status=%d", "knowledge_find", 200)
	if err := tmp.Sync(); err != nil {
		t.Fatalf("sync log file: %v", err)
	}

	content, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "tool=knowledge_find status=200") {
		t.Fatalf("expected log message in file, got %q", string(content))
	}
}

func TestDebugLog_NoFileNoPanic(t *testing.T) {
	prev := debugLogFile
	debugLogFile = nil
	defer func() { debugLogFile = prev }()

	debugLog("this should be ignored")
}

func TestDebugLog_WriteErrorPath(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "tool-exec-closed-*.log")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	_ = tmp.Close()

	prev := debugLogFile
	debugLogFile = tmp
	defer func() { debugLogFile = prev }()

	// Should not panic even when write fails on a closed file descriptor.
	debugLog("should trigger write error branch")
}

func TestInitToolExecutorDebugLog_Disabled(t *testing.T) {
	prev := debugLogFile
	closeDebugLogFileForTest(t)
	defer func() {
		closeDebugLogFileForTest(t)
		debugLogFile = prev
	}()

	t.Setenv("TOOL_EXECUTOR_DEBUG_LOG", "false")
	t.Setenv("TOOL_EXECUTOR_DEBUG_LOG_PATH", "")

	initToolExecutorDebugLog()
	if debugLogFile != nil {
		t.Fatal("expected debugLogFile to remain nil when debug logging is disabled")
	}
}

func TestInitToolExecutorDebugLog_EnabledWithCustomPath(t *testing.T) {
	prev := debugLogFile
	closeDebugLogFileForTest(t)
	defer func() {
		closeDebugLogFileForTest(t)
		debugLogFile = prev
	}()

	path := t.TempDir() + "/tool-exec-debug.log"
	t.Setenv("TOOL_EXECUTOR_DEBUG_LOG", "true")
	t.Setenv("TOOL_EXECUTOR_DEBUG_LOG_PATH", path)

	initToolExecutorDebugLog()
	if debugLogFile == nil {
		t.Fatal("expected debugLogFile to be opened when debug logging is enabled")
	}

	debugLog("hello %s", "world")
	_ = debugLogFile.Sync()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read debug log: %v", err)
	}
	if !strings.Contains(string(content), "hello world") {
		t.Fatalf("expected log content to contain message, got %q", string(content))
	}
}

func TestInitToolExecutorDebugLog_OpenErrorDoesNotPanic(t *testing.T) {
	prev := debugLogFile
	closeDebugLogFileForTest(t)
	defer func() {
		closeDebugLogFileForTest(t)
		debugLogFile = prev
	}()

	// Point path to a directory so os.OpenFile fails with "is a directory".
	t.Setenv("TOOL_EXECUTOR_DEBUG_LOG", "true")
	t.Setenv("TOOL_EXECUTOR_DEBUG_LOG_PATH", t.TempDir())

	initToolExecutorDebugLog()
	if debugLogFile != nil {
		t.Fatal("expected debugLogFile to remain nil when opening log path fails")
	}
}
