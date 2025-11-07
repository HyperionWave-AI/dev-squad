# Hyperion Logging Standards

**Established**: 2025-10-26
**Status**: ✅ ENFORCED (via golangci-lint)
**Scope**: All Go code in `/hyper` directory

---

## Overview

Hyperion enforces structured logging via **zap logger** for all application logging. Direct use of `fmt.Print*`, `log.Print*`, and similar functions is **forbidden** by linting rules.

### Why Structured Logging?

1. **Machine-readable**: JSON format for log aggregation and analysis
2. **Performance**: Zero-allocation logging in production
3. **Consistency**: Unified logging interface across all services
4. **Context**: Key-value pairs for better debugging
5. **Flexibility**: Multiple outputs (console + file) with different formats

---

## Core Principles

### ✅ DO

- **Use zap logger for all logging**
  ```go
  logger.Info("Server started", zap.Int("port", 8080))
  logger.Error("Failed to connect", zap.Error(err))
  ```

- **Use structured fields (key-value pairs)**
  ```go
  logger.Info("User logged in",
      zap.String("userId", userId),
      zap.String("ip", clientIP))
  ```

- **Log at appropriate levels**
  - `Debug`: Detailed diagnostic information
  - `Info`: General informational messages
  - `Warn`: Warning messages for unusual situations
  - `Error`: Error messages with stack traces
  - `Fatal`: Critical errors causing application termination

- **Include context in errors**
  ```go
  logger.Error("Database query failed",
      zap.Error(err),
      zap.String("query", sql),
      zap.String("table", "users"))
  ```

### ❌ DON'T

- **Use fmt.Print* for logging**
  ```go
  // ❌ WRONG - Will fail linting
  fmt.Printf("User %s logged in\n", userId)

  // ✅ CORRECT
  logger.Info("User logged in", zap.String("userId", userId))
  ```

- **Use log.Print* for logging**
  ```go
  // ❌ WRONG - Will fail linting
  log.Printf("Error: %v", err)

  // ✅ CORRECT
  logger.Error("Operation failed", zap.Error(err))
  ```

- **Use string concatenation in logs**
  ```go
  // ❌ WRONG
  logger.Info("User " + userId + " logged in")

  // ✅ CORRECT
  logger.Info("User logged in", zap.String("userId", userId))
  ```

- **Log sensitive data**
  ```go
  // ❌ WRONG - Don't log passwords, tokens, PII
  logger.Info("Login attempt", zap.String("password", password))

  // ✅ CORRECT - Redact sensitive data
  logger.Info("Login attempt", zap.String("userId", userId))
  ```

---

## Zap Logger Quick Reference

### Basic Logging

```go
import "go.uber.org/zap"

// Info level
logger.Info("Operation completed successfully")

// With fields
logger.Info("Request processed",
    zap.String("method", "GET"),
    zap.String("path", "/api/users"),
    zap.Int("status", 200),
    zap.Duration("duration", elapsed))

// Error with stack trace
logger.Error("Database connection failed",
    zap.Error(err),
    zap.String("host", dbHost))

// Fatal (logs and exits)
logger.Fatal("Critical configuration missing",
    zap.String("config", "DATABASE_URL"))
```

### Common Field Types

```go
zap.String("key", "value")           // String values
zap.Int("key", 123)                  // Integer values
zap.Int64("key", 123456789)          // 64-bit integers
zap.Float64("key", 3.14)             // Floating point
zap.Bool("key", true)                // Boolean
zap.Duration("key", time.Second)     // Time duration
zap.Time("key", time.Now())          // Timestamp
zap.Error(err)                       // Error (automatic key "error")
zap.Any("key", complexStruct)        // Any type (uses reflection, slower)
```

### Conditional Logging

```go
// Check if debug logging is enabled before expensive operations
if ce := logger.Check(zap.DebugLevel, "Debug message"); ce != nil {
    ce.Write(
        zap.String("expensiveData", computeExpensiveString()),
    )
}
```

### Logger Initialization

```go
// Development logger (console, colorized, DEBUG level)
logger, _ := zap.NewDevelopment()

// Production logger (JSON, INFO level)
logger, _ := zap.NewProduction()

// Custom configuration (see cmd/coordinator/main.go for example)
func initLogger() (*zap.Logger, error) {
    // See LOGGING_CONFIGURATION.md for full example
}
```

---

## Logging Levels - When to Use

### DEBUG
**When**: Detailed diagnostic information for development/debugging
**Examples**:
- Function entry/exit
- Variable values during execution
- Detailed request/response payloads

```go
logger.Debug("Processing request",
    zap.String("requestId", reqId),
    zap.Any("payload", request))
```

**Note**: Debug logs are excluded from file output (console only) in production.

### INFO
**When**: General informational messages about application flow
**Examples**:
- Server started/stopped
- Configuration loaded
- Successful operations
- Business events

```go
logger.Info("Server started",
    zap.Int("port", 8080),
    zap.String("environment", "production"))
```

### WARN
**When**: Potentially problematic situations that don't stop execution
**Examples**:
- Deprecated features used
- Slow operations
- Retry attempts
- Fallback behavior triggered

```go
logger.Warn("Slow database query detected",
    zap.Duration("elapsed", elapsed),
    zap.String("query", sql))
```

### ERROR
**When**: Error conditions that should be investigated
**Examples**:
- Failed operations
- Exception handling
- Resource exhaustion
- External service failures

```go
logger.Error("Failed to save user",
    zap.Error(err),
    zap.String("userId", userId))
```

**Note**: Error level automatically includes stack traces.

### FATAL
**When**: Critical errors requiring application termination
**Examples**:
- Missing required configuration
- Unable to connect to critical services
- Corrupt data that prevents startup

```go
logger.Fatal("Database connection failed",
    zap.Error(err),
    zap.String("dsn", connectionString))
```

**Warning**: `Fatal` calls `os.Exit(1)` after logging. Use sparingly, only at startup.

---

## Linting Rules

### Enforced by golangci-lint

The `.golangci.yml` configuration enforces logging standards via the **forbidigo** linter:

**Forbidden Patterns**:
- `fmt.Print`
- `fmt.Printf`
- `fmt.Println`
- `log.Print`
- `log.Printf`
- `log.Println`
- `log.Fatal*`
- `log.Panic*`

**Violation Example**:
```
hyper/cmd/coordinator/main.go:63:3: use of `fmt.Printf` forbidden by pattern `^(fmt\.Print(|f|ln)|print|println)$` (forbidigo)
    fmt.Printf("\n")
    ^
```

### Exceptions

The following cases are **excluded** from linting rules:

1. **Utility scripts** (`hyper/scripts/*.go`)
   - Scripts for development/testing may use print statements

2. **Test files** (`*_test.go`)
   - Test output can use print statements for clarity

3. **Interactive prompts** (specific patterns)
   - Dimension mismatch user prompts
   - Critical user decision prompts

**Note**: Even for exceptions, prefer zap logger when possible.

### Running the Linter

```bash
# Check entire codebase
golangci-lint run ./...

# Check specific directory
golangci-lint run ./cmd/coordinator/

# Check specific file
golangci-lint run ./cmd/coordinator/main.go

# Fix auto-fixable issues
golangci-lint run --fix ./...
```

---

## Migration Guide

### Step 1: Identify Violations

Run the linter to find all violations:

```bash
cd /Users/maxmednikov/MaxSpace/hyper/hyper
golangci-lint run ./... | grep forbidigo
```

**Current Status** (as of 2025-10-26):
- `cmd/coordinator/main.go`: 24 violations
- `internal/server/http_server.go`: 3 violations
- `internal/mcp/storage/`: 10 violations
- **Total: 37 violations**

### Step 2: Migrate Print Statements

Replace print statements with appropriate zap logger calls.

#### Example 1: Simple Info Message

**Before**:
```go
fmt.Printf("✓ Loaded configuration from: %s\n", envFile)
```

**After**:
```go
logger.Info("Configuration loaded",
    zap.String("path", envFile))
```

#### Example 2: Error Message

**Before**:
```go
fmt.Printf("Debug: Failed to load %s: %v\n", envFile, err)
```

**After**:
```go
logger.Debug("Failed to load configuration",
    zap.String("path", envFile),
    zap.Error(err))
```

#### Example 3: Warning Message

**Before**:
```go
fmt.Printf("Warning: No .env.hyper found (checked: %s and ./.env.hyper)\n", envFile)
```

**After**:
```go
logger.Warn("Configuration file not found",
    zap.Strings("checkedPaths", []string{envFile, "./.env.hyper"}))
```

#### Example 4: Success Confirmation

**Before**:
```go
fmt.Printf("✅ Collection recreated successfully with %d dimensions\n", expectedDimensions)
```

**After**:
```go
logger.Info("Collection recreated successfully",
    zap.Int("dimensions", expectedDimensions))
```

### Step 3: Interactive Prompts (Special Case)

For legitimate interactive prompts (user input), you may:

1. **Keep fmt.Print for user interaction** (stdin/stdout)
2. **Add zap logger call for audit trail**

**Example**:
```go
// User prompt (allowed via exclude rule)
fmt.Printf("Do you want to recreate the collection? (yes/no): ")
var response string
fmt.Scanln(&response)

// Log the user's choice
logger.Info("User decision recorded",
    zap.String("prompt", "recreate_collection"),
    zap.String("response", response))
```

**Note**: Interactive prompts should only exist in main.go or CLI tools, never in libraries/services.

### Step 4: Verify Migration

After migration, run linter again to verify:

```bash
golangci-lint run ./cmd/coordinator/
```

Expected output: `0 issues` for forbidigo linter.

---

## File Output Configuration

Hyperion uses **dual-output logging**:

### Console Output (Development)
- **Level**: DEBUG and above
- **Format**: Colorized, human-readable
- **Use**: Development, debugging

### File Output (Production)
- **Level**: INFO and above
- **Format**: JSON (machine-readable)
- **Location**: `./logs/coordinator-YYYY-MM-DD.log`
- **Rotation**: Daily (automatic via filename)
- **Use**: Production monitoring, log aggregation

See `LOGGING_CONFIGURATION.md` for full details.

---

## Best Practices

### 1. Use Structured Fields

**Bad**:
```go
logger.Info(fmt.Sprintf("User %s created order %d", userId, orderId))
```

**Good**:
```go
logger.Info("Order created",
    zap.String("userId", userId),
    zap.Int("orderId", orderId))
```

### 2. Group Related Fields

```go
logger.Info("HTTP request completed",
    zap.String("method", r.Method),
    zap.String("path", r.URL.Path),
    zap.Int("status", statusCode),
    zap.Duration("duration", elapsed),
    zap.String("userAgent", r.UserAgent()))
```

### 3. Log Errors with Context

```go
logger.Error("Failed to process payment",
    zap.Error(err),
    zap.String("userId", userId),
    zap.String("paymentId", paymentId),
    zap.Float64("amount", amount))
```

### 4. Use Consistent Message Styles

**Good patterns**:
- Present tense: "Starting server", "Processing request"
- Past tense for completion: "Request completed", "User created"
- Noun phrases: "Database connection failed", "Configuration loaded"

**Avoid**:
- Mixing tenses
- Overly verbose messages
- Implementation details in production logs

### 5. Performance Considerations

For expensive operations, use conditional logging:

```go
// Only compute if debug logging is enabled
if ce := logger.Check(zap.DebugLevel, "Request details"); ce != nil {
    ce.Write(
        zap.Any("headers", r.Header),
        zap.Any("body", parseBody(r)), // Expensive
    )
}
```

---

## Common Patterns

### HTTP Request Logging

```go
logger.Info("HTTP request received",
    zap.String("method", r.Method),
    zap.String("path", r.URL.Path),
    zap.String("remoteAddr", r.RemoteAddr))

// Process request...

logger.Info("HTTP request completed",
    zap.String("method", r.Method),
    zap.String("path", r.URL.Path),
    zap.Int("status", statusCode),
    zap.Duration("duration", time.Since(start)))
```

### Database Operations

```go
logger.Debug("Executing database query",
    zap.String("query", sql),
    zap.Any("params", params))

result, err := db.Exec(sql, params...)
if err != nil {
    logger.Error("Database query failed",
        zap.Error(err),
        zap.String("query", sql))
    return err
}

logger.Info("Database query completed",
    zap.Int64("rowsAffected", result.RowsAffected()))
```

### External API Calls

```go
logger.Info("Calling external API",
    zap.String("service", "payment-gateway"),
    zap.String("endpoint", apiURL))

resp, err := http.Post(apiURL, "application/json", body)
if err != nil {
    logger.Error("External API call failed",
        zap.Error(err),
        zap.String("service", "payment-gateway"),
        zap.String("endpoint", apiURL))
    return err
}

logger.Info("External API call completed",
    zap.String("service", "payment-gateway"),
    zap.Int("statusCode", resp.StatusCode),
    zap.Duration("duration", time.Since(start)))
```

### Background Tasks

```go
logger.Info("Starting background task",
    zap.String("task", "email-sender"))

go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("Background task panicked",
                zap.String("task", "email-sender"),
                zap.Any("panic", r))
        }
    }()

    // Task logic...

    logger.Info("Background task completed",
        zap.String("task", "email-sender"))
}()
```

---

## Troubleshooting

### Issue: Logger not initialized

**Error**:
```go
panic: runtime error: invalid memory address or nil pointer dereference
```

**Cause**: Logger variable is nil

**Solution**:
```go
// Initialize logger first
logger, err := initLogger()
if err != nil {
    log.Fatalf("Failed to initialize logger: %v", err) // Fallback to stdlib
}
defer logger.Sync()

// Now use logger
logger.Info("Application started")
```

### Issue: Logs not appearing in file

**Check**:
1. Log level: File logs start at INFO (DEBUG only in console)
2. Directory exists: `./logs/` created automatically
3. Permissions: File should have 0644 permissions

**Verify**:
```bash
ls -la logs/
tail -f logs/coordinator-$(date +%Y-%m-%d).log
```

### Issue: Linter errors after migration

**Common issues**:
- Forgot to import `go.uber.org/zap`
- Using wrong field type (`zap.String` vs `zap.Int`)
- Unchecked errors from `logger.Sync()`

**Fix**:
```go
import "go.uber.org/zap"

defer func() {
    _ = logger.Sync() // Ignore error in defer
}()
```

---

## Related Documentation

- **Logging Configuration**: `LOGGING_CONFIGURATION.md` - Dual-output setup, file rotation
- **golangci-lint Config**: `.golangci.yml` - Linting rules configuration
- **Zap Documentation**: https://pkg.go.dev/go.uber.org/zap
- **Zap Performance**: https://github.com/uber-go/zap#performance

---

## Summary

### Rules
1. ✅ Use zap logger for all logging
2. ❌ No fmt.Print*, log.Print* (enforced by linter)
3. ✅ Structured logging with key-value fields
4. ✅ Appropriate log levels (Debug, Info, Warn, Error, Fatal)
5. ❌ No sensitive data in logs

### Current Status
- **Linting**: ✅ Configured and enforced
- **File Logging**: ✅ Dual-output (console + file)
- **Migration**: ⏳ 37 violations identified, migration in progress

### Next Steps
1. Migrate remaining 37 fmt.Print violations
2. Add package-level documentation (revive linter)
3. Fix unchecked errors (errcheck linter)
4. Document logging conventions in code comments

---

**Documentation**: `/Users/maxmednikov/MaxSpace/hyper/LOGGING_STANDARDS.md`
**Linter Config**: `/Users/maxmednikov/MaxSpace/hyper/.golangci.yml`
**Status**: ✅ ENFORCED
