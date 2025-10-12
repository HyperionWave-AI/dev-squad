.PHONY: help build native install install-air dev run run-dev run-stdio configure-native run-mcp-local configure-claude-local desktop desktop-dev desktop-build test clean test-connection

# Load environment variables from .env file
include .env
export

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

#
# Build Targets
#

build: native ## Alias for 'native' - build unified hyper binary with embedded UI

native: ## Build native self-contained binary with embedded UI
	@echo "Building unified hyper binary with embedded UI..."
	./build-native.sh
	@echo "✓ Native build complete: bin/hyper"

install: ## Install all dependencies (Go + Node)
	@echo "Installing Go dependencies..."
	cd hyper && go mod download
	@echo "✓ Go dependencies installed"
	@echo "Installing Node.js dependencies..."
	cd coordinator/ui && npm install
	@echo "✓ Node.js dependencies installed"
	@echo "✓ All dependencies installed"

install-air: ## Install Air hot-reload tool locally
	@if command -v air &> /dev/null; then \
		echo "✓ Air already installed (version: $$(air -v))"; \
	else \
		echo "Installing Air..."; \
		go install github.com/air-verse/air@latest; \
		echo "✓ Air installed! Run 'air' in any Go service directory."; \
	fi

#
# Development Targets
#

dev: install-air ## Start development mode with hot reload
	@echo "Starting development mode with hot reload..."
	@if [ ! -f .air.toml ]; then \
		echo "Error: .air.toml not found at project root"; \
		exit 1; \
	fi
	@if [ ! -f .env.hyper ]; then \
		echo "Warning: .env.hyper not found. Using system environment variables."; \
	fi
	./scripts/dev-native.sh

dev-hot: install-air ## Start development with UI hot-reload (Vite dev server + Go Air)
	@echo "Starting development with UI hot-reload..."
	@if [ ! -f .air.toml ]; then \
		echo "Error: .air.toml not found at project root"; \
		exit 1; \
	fi
	./scripts/dev-hot.sh

run: ## Run the native compiled binary (synchronous)
	@echo "Running native binary..."
	@if [ ! -f bin/hyper ]; then \
		echo "Error: Native binary not found. Run 'make native' first."; \
		exit 1; \
	fi
	@if [ ! -f .env.native ]; then \
		echo "Warning: .env.native not found. Please configure environment variables."; \
		echo "Copy .env.native to your project root and update with your settings."; \
	fi
	./bin/hyper --mode=http

run-dev: ## Run with Air hot-reload (unified hyper binary)
	@echo "Starting development mode with hot-reload..."
	@echo "Using Air for automatic rebuild on file changes"
	@if ! command -v air &> /dev/null; then \
		echo "Error: Air not found. Install with 'make install-air'"; \
		exit 1; \
	fi
	@if [ ! -f .air.toml ]; then \
		echo "Error: .air.toml not found at project root"; \
		exit 1; \
	fi
	@echo "Building and running unified hyper binary with Air..."
	air

run-stdio: ## Run the native binary in stdio mode (for Claude Code/MCP)
	@echo "Running native binary in stdio mode..."
	@if [ ! -f bin/hyper ]; then \
		echo "Error: Native binary not found. Run 'make native' first."; \
		exit 1; \
	fi
	@if [ ! -f .env.native ]; then \
		echo "Warning: .env.native not found. Using system environment variables."; \
	fi
	./bin/hyper --mode=mcp

#
# MCP Configuration Targets
#

configure-native: native ## Configure Claude Code to use native binary (stdio mode)
	@echo "🚀 Configuring Claude Code to use native binary..."
	@echo ""
	@if [ ! -f bin/hyper ]; then \
		echo "Error: Native binary not found. Run 'make native' first."; \
		exit 1; \
	fi
	@echo "Removing old hyper configuration (if exists)..."
	@claude mcp remove hyper --scope user 2>/dev/null || true
	@claude mcp remove hyper --scope project 2>/dev/null || true
	@echo ""
	@echo "Adding hyper native binary (stdio mode, user scope)..."
	@claude mcp add hyper "$(shell pwd)/bin/hyper" --args "--mode=mcp" --scope user
	@echo ""
	@echo "✅ Configuration complete!"
	@echo "Native binary: $(shell pwd)/bin/hyper"
	@echo "Mode: stdio (MCP protocol)"
	@echo "Config file: .env.native (place in project root)"
	@echo ""
	@echo "Verify connection:"
	@claude mcp list 2>&1 | grep hyper || echo "❌ Failed to configure"

run-mcp-http: native ## Run unified binary in HTTP mode (REST API + UI on port 7095)
	@echo "Starting unified hyper binary in HTTP mode..."
	@echo "REST API: http://localhost:7095/api/v1"
	@echo "Web UI: http://localhost:7095"
	@echo "Health: http://localhost:7095/api/v1/health"
	@if [ ! -f bin/hyper ]; then \
		echo "Error: Native binary not found. Run 'make native' first."; \
		exit 1; \
	fi
	./bin/hyper --mode=http

#
# Desktop App Targets
#

desktop: ## Build and run desktop app (development mode)
	@echo "🖥️  Starting Hyperion Coordinator Desktop App (development)..."
	@if [ ! -f bin/hyper ]; then \
		echo "Building native binary first..."; \
		$(MAKE) native; \
	fi
	@if [ ! -f .env.hyper ]; then \
		echo "⚠️  Warning: .env.hyper not found. Please configure before running."; \
	fi
	@echo "Starting hyper server in background..."
	@pkill -f "bin/hyper" || true
	@sleep 1
	@cd bin && ./hyper --mode=http 2>&1 | sed 's/^/[hyper] /' &
	@echo "Waiting for server to be ready..."
	@sleep 5
	@echo "Launching Tauri desktop app..."
	cd desktop-app && npm install && npm run dev

desktop-dev: desktop ## Alias for desktop (development mode)

desktop-build: native ## Build desktop app for distribution
	@echo "🖥️  Building Hyperion Coordinator Desktop App for distribution..."
	@echo "This will create a native app bundle for your platform"
	@if [ ! -f .env.hyper ]; then \
		echo "⚠️  Warning: .env.hyper not found. The app will need it at runtime."; \
	fi
	cd desktop-app && npm install && npm run build
	@echo ""
	@echo "✅ Desktop app built successfully!"
	@echo ""
	@echo "📦 Output location:"
	@echo "   macOS:   desktop-app/src-tauri/target/release/bundle/dmg/"
	@echo "   macOS:   desktop-app/src-tauri/target/release/bundle/macos/"
	@echo "   Windows: desktop-app/src-tauri/target/release/bundle/msi/"
	@echo "   Linux:   desktop-app/src-tauri/target/release/bundle/appimage/"
	@echo ""
	@echo "💡 To install:"
	@echo "   macOS:   Open the .dmg file"
	@echo "   Windows: Run the .msi installer"
	@echo "   Linux:   Run the .AppImage file"

#
# Utilities
#

test: ## Run tests
	@echo "Running hyper tests..."
	cd hyper && go test ./...
	@echo "✓ All tests passed"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/hyper || true
	@rm -rf hyper/bin/ || true
	@rm -rf coordinator/ui/dist || true
	@rm -rf hyper/embed/ui || true
	@echo "✓ Clean complete (node_modules preserved)"
