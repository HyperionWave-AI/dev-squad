# Hyperion Unified Coordinator Makefile
# Architecture: UI → REST API → Storage Layer

.PHONY: help build build-ui build-coordinator run dev test clean docker-build docker-up docker-down docker-logs install

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=coordinator
BINARY_PATH=bin/$(BINARY_NAME)
UI_DIST=ui/dist
DOCKER_COMPOSE=docker-compose -f ../docker-compose.unified.yml
GO_FILES=$(shell find . -name '*.go' -not -path "./ui/*" -not -path "./vendor/*")

## help: Display this help message
help:
	@echo "Hyperion Unified Coordinator - Build & Development Commands"
	@echo ""
	@echo "Architecture: UI → REST API → Storage Layer"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'

## build: Build coordinator binary and UI
build: build-ui build-coordinator
	@echo "✅ Build complete!"
	@echo "   Binary: $(BINARY_PATH) ($(shell du -h $(BINARY_PATH) 2>/dev/null | cut -f1))"
	@echo "   UI: $(UI_DIST)"

## build-ui: Build production UI bundle
build-ui:
	@echo "🎨 Building UI..."
	@cd ui && npm install && npm run build
	@echo "✅ UI built: $(UI_DIST)"

## build-coordinator: Build coordinator binary
build-coordinator:
	@echo "🔨 Building coordinator binary..."
	@mkdir -p bin
	@go build -o $(BINARY_PATH) ./cmd/coordinator
	@echo "✅ Coordinator built: $(BINARY_PATH)"

## run: Run coordinator in production mode (requires built UI)
run: build
	@echo "🚀 Starting unified coordinator..."
	@./$(BINARY_PATH) --mode=http

## dev: Run coordinator in native development mode with hot-reload (no Docker)
dev:
	@./scripts/dev-native.sh

## dev-docker: Run coordinator in Docker development mode with hot-reload
dev-docker:
	@echo "🔥 Starting development mode (Docker)..."
	@echo "   - Coordinator: Air hot-reload"
	@echo "   - UI: Vite dev server"
	@$(DOCKER_COMPOSE) up --build

## dev-coordinator: Run only coordinator with Air hot-reload
dev-coordinator:
	@echo "🔥 Starting coordinator with Air hot-reload..."
	@cd cmd/coordinator && air

## dev-ui: Run only UI with Vite dev server
dev-ui:
	@echo "🎨 Starting UI dev server..."
	@cd ui && npm run dev

## test: Run all tests
test: test-go test-ui
	@echo "✅ All tests passed"

## test-go: Run Go tests
test-go:
	@echo "🧪 Running Go tests..."
	@go test -v ./...

## test-ui: Run UI tests
test-ui:
	@echo "🧪 Running UI tests..."
	@cd ui && npm run test

## clean: Remove build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BINARY_PATH) bin/
	@rm -rf $(UI_DIST)
	@rm -rf ui/node_modules
	@rm -rf tmp/
	@echo "✅ Clean complete"

## docker-build: Build Docker images
docker-build:
	@echo "🐳 Building Docker images..."
	@docker build -t hyperion-coordinator:latest -f Dockerfile .
	@docker build -t hyperion-coordinator-dev:latest -f Dockerfile.dev .
	@docker build -t hyperion-ui:latest -f ui/Dockerfile ui/
	@echo "✅ Docker images built"

## docker-up: Start services with Docker Compose
docker-up:
	@echo "🐳 Starting Docker Compose services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "✅ Services started"
	@echo ""
	@echo "📖 Open: http://localhost:7777"
	@echo "🔍 API: http://localhost:7095/api"
	@echo ""
	@echo "View logs: make docker-logs"
	@echo "Stop: make docker-down"

## docker-down: Stop Docker Compose services
docker-down:
	@echo "🛑 Stopping Docker Compose services..."
	@$(DOCKER_COMPOSE) down
	@echo "✅ Services stopped"

## docker-logs: View Docker Compose logs
docker-logs:
	@$(DOCKER_COMPOSE) logs -f

## docker-restart: Restart Docker Compose services
docker-restart: docker-down docker-up

## install: Install dependencies (Go modules + npm packages)
install:
	@echo "📦 Installing dependencies..."
	@go mod download
	@cd ui && npm install
	@echo "✅ Dependencies installed"

## lint: Run linters
lint:
	@echo "🔍 Running linters..."
	@golangci-lint run ./...
	@cd ui && npm run lint
	@echo "✅ Linting complete"

## format: Format code
format:
	@echo "📝 Formatting code..."
	@gofmt -s -w $(GO_FILES)
	@cd ui && npm run format
	@echo "✅ Formatting complete"

## prod-deploy: Build and deploy to production
prod-deploy: build docker-build
	@echo "🚀 Deploying to production..."
	@echo "   1. Build complete"
	@echo "   2. Docker images ready"
	@echo "   3. Push images: docker push ..."
	@echo "   4. Deploy: kubectl apply ..."
	@echo ""
	@echo "⚠️  Manual deployment steps required"

## db-reset: Reset local database (WARNING: deletes all data)
db-reset:
	@echo "⚠️  WARNING: This will delete ALL local database data!"
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "🗑️  Resetting database..."; \
		$(DOCKER_COMPOSE) down -v; \
		$(DOCKER_COMPOSE) up -d mongodb qdrant; \
		echo "✅ Database reset complete"; \
	else \
		echo "❌ Cancelled"; \
	fi

## status: Show service status
status:
	@echo "📊 Service Status"
	@echo ""
	@if [ -f $(BINARY_PATH) ]; then \
		echo "✅ Binary: $(BINARY_PATH) ($(shell du -h $(BINARY_PATH) 2>/dev/null | cut -f1))"; \
	else \
		echo "❌ Binary: Not built"; \
	fi
	@if [ -d $(UI_DIST) ]; then \
		echo "✅ UI: Built"; \
	else \
		echo "❌ UI: Not built"; \
	fi
	@echo ""
	@echo "Docker Containers:"
	@$(DOCKER_COMPOSE) ps 2>/dev/null || echo "  (not running)"

## version: Show version information
version:
	@echo "Hyperion Unified Coordinator"
	@echo ""
	@echo "Go version: $(shell go version)"
	@echo "Node version: $(shell node --version)"
	@echo "npm version: $(shell npm --version)"
	@echo "Docker version: $(shell docker --version)"

## quick-start: Quick start for development (build + run)
quick-start: build
	@echo ""
	@echo "🎉 Quick start complete!"
	@echo ""
	@echo "Starting coordinator..."
	@./start-coordinator.sh
