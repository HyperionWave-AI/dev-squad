.PHONY: build build-hyper clean test install-tools dev

# Build unified hyper binary
build: build-hyper

# Build the unified hyper binary (coordinator with all features)
build-hyper:
	@echo "Building unified hyper binary..."
	@mkdir -p bin
	@go build -o bin/hyper ./cmd/coordinator
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
