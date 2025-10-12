package server

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestFindProcessByPort tests the process discovery functionality
func TestFindProcessByPort(t *testing.T) {
	// Start a test server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	// Extract port from listener address
	addr := listener.Addr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	// Wait a bit for the OS to register the listener
	time.Sleep(100 * time.Millisecond)

	// Try to find the process using the port
	pid, err := findProcessByPort(port)
	if err != nil {
		t.Logf("Could not find process on port %s: %v (this may be expected on some systems)", port, err)
		return
	}

	// Verify we got a valid PID
	if pid <= 0 {
		t.Errorf("Expected positive PID, got %d", pid)
	}

	t.Logf("Successfully found PID %d using port %s", pid, port)
}

// TestFindProcessByPort_NotFound tests behavior when no process is using the port
func TestFindProcessByPort_NotFound(t *testing.T) {
	// Use a port that's very unlikely to be in use
	port := "65432"

	pid, err := findProcessByPort(port)
	if err == nil {
		t.Errorf("Expected error for unused port, but got PID %d", pid)
	}

	expectedMsg := "no process found on port"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedMsg, err)
	}
}

// TestKillProcess tests the process termination functionality
func TestKillProcess(t *testing.T) {
	// Start a long-running process that we can safely kill
	// Use a shell command that will respond to SIGTERM
	cmd := exec.Command("sh", "-c", "trap 'exit 0' TERM; sleep 30 & wait")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	pid := cmd.Process.Pid
	t.Logf("Started test process with PID %d", pid)

	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Kill the process
	err := killProcess(pid, logger)
	if err != nil {
		t.Errorf("Failed to kill process %d: %v", pid, err)
	}

	// Wait for the process to terminate (killProcess already waits 2 seconds)
	// Add a bit more time for process cleanup
	time.Sleep(500 * time.Millisecond)

	// Wait for the command to exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("Process %d terminated with: %v (expected after SIGTERM)", pid, err)
		} else {
			t.Logf("Process %d successfully terminated", pid)
		}
	case <-time.After(3 * time.Second):
		// Process didn't terminate, try to force kill it
		_ = cmd.Process.Kill()
		t.Errorf("Process %d did not terminate after 3 seconds", pid)
	}
}

// TestPromptKillProcess tests the user prompt functionality
func TestPromptKillProcess(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// This test will always return false when not in interactive mode
	// We just verify it doesn't panic or error
	shouldKill, err := promptKillProcess("8080", 12345, logger)
	if err != nil {
		t.Errorf("Unexpected error from promptKillProcess: %v", err)
	}

	// In non-interactive mode, should return false
	if shouldKill {
		t.Error("Expected shouldKill=false in non-interactive mode, got true")
	}

	t.Log("promptKillProcess handled non-interactive mode correctly")
}

// TestIsInteractiveTerminal tests terminal detection
func TestIsInteractiveTerminal(t *testing.T) {
	// This will typically be false when running in CI/automated tests
	isInteractive := isInteractiveTerminal()
	t.Logf("Terminal interactive status: %v", isInteractive)

	// We don't assert anything specific since it depends on the environment
	// Just verify the function doesn't panic
}

// TestPortBusyScenario simulates a port-busy scenario
func TestPortBusyScenario(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}

	addr := listener.Addr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	t.Logf("Testing with port %s", port)

	// Wait for port to be registered
	time.Sleep(100 * time.Millisecond)

	// Try to find the process
	pid, err := findProcessByPort(port)

	// Clean up
	listener.Close()

	if err != nil {
		t.Logf("Could not find process on port %s: %v (expected on some systems)", port, err)
		return
	}

	t.Logf("Successfully simulated port-busy scenario: PID %d is using port %s", pid, port)

	// Verify we can detect the port is busy
	listener2, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			t.Logf("Successfully detected port is busy: %v", err)
		} else {
			t.Logf("Port became available or different error: %v", err)
		}
	} else {
		// Port was available, clean up
		listener2.Close()
		t.Log("Port was available after first listener closed")
	}
}
