# Makefile Build and Deploy Patterns

## Overview

Standard Makefile targets for building, testing, and deploying Hyperion services with support for both development and production workflows.

## Technology

- GNU Make
- Go 1.25
- Vite (UI development)
- Air (Go hot-reload)
- Docker/Kubernetes

## Use Case

Use these Makefile targets for consistent build, test, and deployment operations across the Hyperion project. The Makefile provides unified commands for both backend and frontend development workflows.

## Implementation

**File Reference**: `Makefile:1-227`

### Key Build Targets

```makefile
# Build unified binary with embedded UI
make native
# → ./build-native.sh
# → Output: bin/hyper (17MB)

# Full-stack hot-reload development
make dev-hot
# → Vite dev server (UI) + Air (Go backend)
# → Auto-rebuilds on file changes

# Go backend hot-reload only
make dev
# → ./scripts/dev-native.sh
# → Air watches *.go files

# Run tests
make test
# → cd hyper && go test ./...
```

### Development Workflow

```makefile
# 1. Install dependencies
make install
# → go mod download + npm install

# 2. Start development
make dev-hot
# → Full-stack with hot-reload

# 3. Run tests
make test

# 4. Build for production
make native
# → Creates bin/hyper binary
```

### Configuration Targets

```makefile
# Configure Claude Code MCP
make configure-native
# → claude mcp add hyper ./bin/hyper --args "--mode=mcp"

# Run in HTTP mode
make run-mcp-http
# → ./bin/hyper --mode=http
# → REST API: http://localhost:7095
```

### Clean Targets

```makefile
# Clean build artifacts only
make clean
# → Removes bin/, build/, dist/

# Deep clean (includes node_modules)
make clean-all
# → Removes all generated files and dependencies
```

## Key Points

### Makefile Patterns

1. **.PHONY**: All targets are phony (no file dependencies)
   ```makefile
   .PHONY: native dev test clean
   ```

2. **include .env**: Auto-loads environment variables
   ```makefile
   -include .env
   export
   ```

3. **@echo**: Suppresses command echo, shows friendly messages
   ```makefile
   @echo "Building native binary..."
   @./build-native.sh
   ```

4. **|| true**: Prevents errors from stopping execution
   ```makefile
   @rm -rf bin/ || true
   ```

5. **Conditional checks**: Validates prerequisites before running
   ```makefile
   @command -v go >/dev/null 2>&1 || { echo "Go not found"; exit 1; }
   ```

### Development Targets

**Full-Stack Development** (`make dev-hot`):
- Starts Vite dev server on port 5173
- Starts Air-watched Go backend
- Hot-reload for both frontend and backend
- Ideal for UI and API development

**Backend-Only Development** (`make dev`):
- Only starts Air-watched Go backend
- Faster startup for backend-only work
- Uses embedded UI from previous build

**Testing** (`make test`):
- Runs Go test suite
- Includes all packages in `hyper/`
- Use `make test-verbose` for detailed output

### Production Targets

**Native Binary Build** (`make native`):
- Builds frontend with Vite (optimized)
- Embeds UI into Go binary
- Compiles single binary (~17MB)
- Output: `bin/hyper`

**Production Build by Service** (`make prod-build`):
```bash
make prod-build SERVICE=coordinator
# → Builds specific service Docker image
```

**Quick Deploy** (`make prod-quick`):
```bash
make prod-quick SERVICE=coordinator
# → Build + deploy (dev environment only)
```

### Environment Variables

Required in `.env`:
```bash
MONGODB_URI=mongodb://localhost:27017
QDRANT_URL=http://localhost:6333
TEI_URL=http://localhost:8080
ANTHROPIC_API_KEY=sk-...
```

### Best Practices

1. **Use Makefile for all operations**: Don't run scripts directly
2. **Development**: Start with `make dev-hot` for full-stack work
3. **Testing**: Run `make test` before committing
4. **Production builds**: Use `make native` for local binaries
5. **Clean builds**: Run `make clean` if build behaves unexpectedly
6. **MCP setup**: Use `make configure-native` to register with Claude Code

### Common Workflows

**New Developer Setup**:
```bash
make install          # Install dependencies
make dev-hot          # Start development
```

**Feature Development**:
```bash
make dev-hot          # Develop with hot-reload
make test             # Run tests
make native           # Build production binary
```

**Troubleshooting**:
```bash
make clean-all        # Remove all generated files
make install          # Reinstall dependencies
make native           # Rebuild from scratch
```

### Target Dependencies

```makefile
native: clean install build-ui build-go
dev-hot: install
test: install
clean-all: clean
```

## Metadata

- **Domain**: build-tools
- **Language**: makefile
- **Pattern**: build-deploy
- **Technology**: make, go, vite
