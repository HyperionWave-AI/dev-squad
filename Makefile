.PHONY: help build native install install-air dev run run-dev run-stdio configure-native run-mcp-local configure-claude-local desktop desktop-dev desktop-build desktop-build-all desktop-install desktop-build-install desktop-install-applications test clean test-connection

# Load environment variables from .env file
include .env
export

MODE ?= dev
PLATFORM ?=
PLATFORMS ?=

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

init: native ## Initialize a new Hyperion project (creates docker-compose.yml + .env.hyper)
	@echo "Running hyper init..."
	@if [ ! -f bin/hyper ]; then \
		echo "Error: Native binary not found. Run 'make native' first."; \
		exit 1; \
	fi
	./bin/hyper init

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
	cd ui && npm install
	@echo "✓ Node.js dependencies installed"
	@echo "✓ All dependencies installed"

install-air: ## Install Air hot-reload tool locally
	@if command -v air > /dev/null; then \
		echo "✓ Air already installed (version: $$(air -v))"; \
	else \
		echo "Installing Air..."; \
		go install github.com/air-verse/air@latest; \
		echo "✓ Air installed! Run 'air' in any Go service directory."; \
	fi

#
# Development Targets
#

dev: install-air ## Start Go backend hot reload only (no UI compilation)
	@echo "Starting Go backend development mode with hot reload..."
	@if [ ! -f .air.toml ]; then \
		echo "Error: .air.toml not found at project root"; \
		exit 1; \
	fi
	@if [ ! -f .env.hyper ]; then \
		echo "Warning: .env.hyper not found. Using system environment variables."; \
	fi
	./scripts/dev-native.sh

dev-hot: install-air ## Start full-stack hot-reload (Vite dev server + Go Air)
	@echo "Starting full-stack development with hot-reload..."
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
		echo "Warning: .env.native nogit  t found. Please configure environment variables."; \
		echo "Copy .env.native to your project root and update with your settings."; \
	fi
	./bin/hyper --mode=http --config=.env.native

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

desktop: ## Tauri desktop runner (MODE=dev|build, PLATFORM=<one>, PLATFORMS="<many>")
	@echo "🖥️  Hyper Desktop"
	@if [ ! -d desktop-app/src-tauri ]; then \
		echo "Error: desktop-app/src-tauri not found."; \
		exit 1; \
	fi
	@if [ ! -f .env.hyper ]; then \
		echo "⚠️  Warning: .env.hyper not found. Hyper sidecar may fail to start."; \
	fi
	@./scripts/desktop.sh --mode "$(MODE)" $(if $(PLATFORM),--platform "$(PLATFORM)") $(if $(PLATFORMS),--platforms "$(PLATFORMS)")

desktop-dev: ## Launch desktop app in development mode (host platform)
	@$(MAKE) desktop MODE=dev

desktop-build: ## Build desktop bundle(s); accepts PLATFORM or PLATFORMS
	@$(MAKE) desktop MODE=build PLATFORM="$(PLATFORM)" PLATFORMS="$(PLATFORMS)"

desktop-build-all: ## Build common platform bundles (requires cross toolchains where applicable)
	@$(MAKE) desktop MODE=build PLATFORMS="macos-arm64 macos-amd64 linux-amd64 windows-amd64"

desktop-install: ## Install desktop app (host by default). Supports PLATFORM and INSTALL_DEST
	@./scripts/desktop.sh --mode install $(if $(PLATFORM),--platform "$(PLATFORM)") $(if $(PLATFORMS),--platforms "$(PLATFORMS)")

desktop-build-install: ## Build desktop app, then install to /Applications (macOS)
	@$(MAKE) desktop-build PLATFORM="$(PLATFORM)"
	@$(MAKE) desktop-install PLATFORM="$(PLATFORM)" INSTALL_DEST="/Applications"

desktop-install-applications: desktop-build-install ## Alias: build + install desktop app to /Applications

#
# Utilities
#

test: ## Run tests
	@echo "Running hyper tests..."
	cd hyper && go test ./...
	@echo "✓ All tests passed"

clean: ## Clean build artifacts (keeps node_modules)
	@echo "Cleaning build artifacts..."
	@rm -rf bin/hyper || true
	@rm -rf bin/hyper2 || true
	@rm -rf hyper/bin/ || true
	@rm -rf ui/dist || true
	@rm -rf ui2/dist || true
	@rm -rf hyper/embed/ui || true
	@rm -rf hyper/embed/ui2 || true
	@echo "✓ Clean complete (node_modules preserved)"

clean-all: ## Clean everything including node_modules and Go cache
	@echo "⚠️  This will remove node_modules and Go cache"
	@read -p "Continue? (yes/no): " confirm && [ "$$confirm" = "yes" ] || exit 1
	@echo "Cleaning all artifacts..."
	@rm -rf bin/ || true
	@rm -rf hyper/bin/ || true
	@rm -rf ui/dist ui/node_modules || true
	@rm -rf ui2/dist ui2/node_modules || true
	@rm -rf hyper/embed/ui hyper/embed/ui2 || true
	@echo "Cleaning Go cache..."
	@cd hyper && go clean -modcache || true
	@echo "✓ Deep clean complete"

clean-install: ## Run clean install script (interactive)
	@./clean-install.sh
