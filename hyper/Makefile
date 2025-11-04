.PHONY: build build-hyper clean test install-tools dev

# Build unified hyper binary
build: build-hyper

# Build the unified hyper binary (coordinator with all features)
build-hyper:
	@echo "Building unified hyper binary..."
	@mkdir -p bin
	@BUILD_TIME=$$(date -u '+%Y-%m-%d %H:%M:%S UTC') && \
	GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
	go build -ldflags "-X 'main.BuildTime=$$BUILD_TIME' -X 'main.GitCommit=$$GIT_COMMIT'" -o bin/hyper ./cmd/coordinator
	@echo "✓ Build complete: bin/hyper"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v -cover
	@echo "✓ Tests complete"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/air-verse/air@latest
	@echo "✓ Air installed"

# Run with hot reload (requires air)
dev:
	@echo "Starting unified hyper binary with hot reload..."
	@if ! command -v air &> /dev/null; then \
		echo "Error: Air not found. Install with 'make install-tools'"; \
		exit 1; \
	fi
	@air

# Install Go dependencies
install:
	@echo "Installing Go dependencies..."
	@go mod download
	@echo "✓ Dependencies installed"

# Run with hot reload for ui2 development (port 5177)
dev-hot2:
	@echo "Starting hyper for ui2 development (port 5177)..."
	@if [ ! -f .env.hyper.hot2 ]; then \
		echo "Error: .env.hyper.hot2 config file not found"; \
		exit 1; \
	fi
	@cp .env.hyper.hot2 .env.hyper.hot
	@if command -v air &> /dev/null; then \
		echo "Using air for hot-reload..."; \
		air; \
	else \
		echo "Air not found, running binary directly (no hot-reload)..."; \
		make build-hyper && ./bin/hyper -config .env.hyper.hot2 -mode http; \
	fi
