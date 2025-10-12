# Port-Busy Recovery Feature

## Overview

The Hyperion HTTP server now includes an interactive port-busy recovery feature that automatically detects when the configured port is already in use and offers to kill the conflicting process.

## Problem Statement

Previously, when the HTTP server failed to start because the port was busy, users would see:
```
listen tcp :7095: bind: address already in use
```

Users had to manually:
1. Find the process using the port: `lsof -ti tcp:7095`
2. Kill the process: `kill <PID>`
3. Restart the server

## Solution

The new port-busy recovery feature:
1. **Detects** the "address already in use" error
2. **Finds** the process using the port (PID)
3. **Prompts** the user: "Port 7095 is busy. Kill the process (PID: XXXX)? [y/N]"
4. **Kills** the process if user confirms (y)
5. **Retries** starting the server automatically
6. **Fails gracefully** if user declines (N) with instructions

## Usage

### Interactive Mode (Terminal)

When running in an interactive terminal:

```bash
$ ./bin/hyper --mode=http

⚠️  Port 7095 is already in use by process 12345
Kill the process and retry? [y/N]: y
✓ Killed process 12345, retrying... (attempt 1/3)
HTTP server listening on :7095
```

### Non-Interactive Mode (Daemon/CI)

When running as a daemon or in CI/CD:
- The prompt is skipped automatically
- The server fails immediately with clear instructions
- No user interaction required

```bash
Port 7095 is busy (PID: 12345). Please kill the process manually:
  kill 12345
  or use:
  lsof -ti tcp:7095 | xargs kill
```

## Configuration

### Retry Behavior

- **Max Retries**: 3 attempts
- **Wait Time**: 1 second between retries
- **Timeout**: 500ms to detect immediate startup failures

### Supported Platforms

- ✅ **macOS**: Uses `lsof -ti tcp:<port>`
- ✅ **Linux**: Fallback to `netstat` if `lsof` not available
- ⚠️ **Windows**: Not currently supported (needs `netstat` port detection)

## Implementation Details

### Helper Functions

#### `findProcessByPort(port string) (int, error)`
Finds the PID of the process using a specific port.
- **Primary**: `lsof -ti tcp:<port>` (macOS/BSD)
- **Fallback**: `netstat` parsing (Linux)

#### `isInteractiveTerminal() bool`
Checks if the program is running in an interactive terminal using `golang.org/x/term`.

#### `promptKillProcess(port string, pid int, logger *zap.Logger) (bool, error)`
Prompts the user to kill the process. Only shows prompt in interactive mode.

#### `killProcess(pid int, logger *zap.Logger) error`
Sends SIGTERM to the process and waits 2 seconds for graceful shutdown.

### Retry Logic

```go
maxRetries := 3
for attempt := 1; attempt <= maxRetries; attempt++ {
    // Start server in goroutine
    go func() {
        if err := srv.ListenAndServe(); err != nil {
            serverStarted <- err
        }
    }()

    // Wait briefly to detect immediate failures
    select {
    case startErr = <-serverStarted:
        if strings.Contains(startErr.Error(), "bind: address already in use") {
            // Port busy - find and kill process
            pid, _ := findProcessByPort(port)
            shouldKill, _ := promptKillProcess(port, pid, logger)
            if shouldKill {
                killProcess(pid, logger)
                // Retry
                continue
            }
        }
        return startErr
    case <-time.After(500 * time.Millisecond):
        // Server started successfully
        break
    }
}
```

## Testing

### Unit Tests

Run the comprehensive test suite:

```bash
go test -v ./internal/server
```

Tests include:
- `TestFindProcessByPort`: Process discovery
- `TestFindProcessByPort_NotFound`: Handling non-existent processes
- `TestKillProcess`: SIGTERM handling
- `TestPromptKillProcess`: Interactive prompt (non-interactive mode)
- `TestIsInteractiveTerminal`: Terminal detection
- `TestPortBusyScenario`: End-to-end port-busy simulation

### Manual Testing

Use the provided test script:

```bash
./test_port_busy.sh
```

Or manually:
1. Start a process on port 7095:
   ```bash
   python3 -m http.server 7095
   ```

2. In another terminal, start hyper:
   ```bash
   ./bin/hyper --mode=http
   ```

3. You should see the interactive prompt.

## Error Handling

### Permission Denied

If the process is owned by another user:
```
failed to kill process 12345: failed to terminate process: operation not permitted
```

**Solution**: Run with `sudo` or kill the process manually as the owning user.

### Process Not Found

If the port is busy but no process can be found:
```
Port 7095 is busy but couldn't find the process: no process found on port 7095
Manually kill the process with:
  lsof -ti tcp:7095 | xargs kill
```

**Solution**: The port may be used by a system service or the detection failed. Kill manually.

### User Declines to Kill Process

If user types 'N' or just presses Enter:
```
Port 7095 is busy (PID: 12345). Please kill the process manually:
  kill 12345
  or use:
  lsof -ti tcp:7095 | xargs kill
```

**Solution**: Follow the instructions to kill the process manually.

## Dependencies

- `golang.org/x/term`: Terminal detection
- Standard library: `os/exec`, `syscall`, `bufio`

## Security Considerations

1. **SIGTERM Only**: Uses graceful SIGTERM, not SIGKILL
2. **User Confirmation**: Always prompts before killing (interactive mode)
3. **No Privilege Escalation**: Cannot kill processes owned by other users without sudo
4. **Non-Interactive Safety**: Skips prompt in daemon/CI mode to prevent hangs

## Future Enhancements

- [ ] Support Windows port detection (`netstat -ano`)
- [ ] Add SIGKILL fallback if SIGTERM fails
- [ ] Allow configuring max retries via environment variable
- [ ] Support killing multiple processes on the same port
- [ ] Add metrics/logging for port-busy occurrences

## Troubleshooting

### Issue: Prompt doesn't appear

**Cause**: Running in non-interactive mode (daemon, CI, pipe)

**Solution**: The prompt is intentionally skipped. Kill the process manually:
```bash
kill $(lsof -ti tcp:7095)
```

### Issue: Process still running after kill

**Cause**: Process didn't respond to SIGTERM within 2 seconds

**Solution**: Force kill with SIGKILL:
```bash
kill -9 <PID>
```

### Issue: "lsof: command not found"

**Cause**: `lsof` not installed (rare on macOS, possible on Linux)

**Solution**: Install lsof:
```bash
# Ubuntu/Debian
sudo apt-get install lsof

# Fedora/RHEL
sudo yum install lsof
```

## References

- Implementation: `hyper/internal/server/http_server.go`
- Tests: `hyper/internal/server/port_recovery_test.go`
- Related Issue: Port already in use error in HTTP server startup
