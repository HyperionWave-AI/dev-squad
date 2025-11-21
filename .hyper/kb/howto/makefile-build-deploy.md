# How to Use Makefile Commands for Building and Deploying

**Collection:** howto
**Tags:** makefile, build, deploy, automation, devops
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide explains how to use Makefile commands for building, testing, and deploying Hyperion services. Makefiles provide a standardized interface for common development and deployment tasks across the entire codebase.

## Prerequisites

- GNU Make installed (`make --version`)
- Understanding of Makefile syntax
- Familiarity with project structure

## When to Use This Guide

- Building Go services and embedding UI
- Running local development servers
- Executing tests and linters
- Deploying to development/production environments
- Understanding Hyperion's build pipeline

**Important:** Production deploys MUST use CI/CD (GitHub Actions). Never deploy directly to production via local commands.

---

## Common Makefile Targets

### Development Targets

#### `make help`
Display all available targets with descriptions:
```bash
make help
```

Output:
```
Available targets:
  native              Build unified hyper binary with embedded UI
  dev                 Start Go backend hot reload
  dev-hot             Start full-stack hot-reload (Vite + Go Air)
  run                 Run the native compiled binary
  test                Run tests
  clean               Clean build artifacts
```

#### `make native`
Build unified binary with embedded UI:
```bash
make native
```

**What it does:**
1. Builds React frontend (`ui/`) using Vite
2. Embeds static files into Go binary
3. Compiles Go service with embedded UI
4. Outputs to `bin/hyper`

**Use when:**
- Preparing for deployment
- Testing production build locally
- Creating distributable binary

#### `make dev`
Start backend with hot reload (Air):
```bash
make dev
```

**What it does:**
1. Installs Air if not present
2. Watches Go files for changes
3. Rebuilds and restarts on file save
4. Does NOT rebuild UI

**Use when:**
- Developing backend-only features
- Testing API endpoints
- UI already built

#### `make dev-hot`
Start full-stack with hot reload:
```bash
make dev-hot
```

**What it does:**
1. Starts Vite dev server (port 5173)
2. Starts Go backend with Air
3. Watches both frontend and backend
4. Hot-reloads on any file change

**Use when:**
- Developing UI and backend simultaneously
- Full-stack feature development
- Maximum development speed

### Build Targets

#### `make install`
Install all project dependencies:
```bash
make install
```

**What it does:**
1. `go mod download` - Go dependencies
2. `npm install` in `ui/` - Node dependencies

#### `make install-air`
Install Air hot-reload tool:
```bash
make install-air
```

**What it does:**
- Installs `github.com/air-verse/air` globally
- Verifies installation

### Testing Targets

#### `make test`
Run all tests:
```bash
make test
```

**What it does:**
```bash
cd hyper && go test ./...
```

**Options:**
```bash
# Verbose output
cd hyper && go test -v ./...

# With race detector
cd hyper && go test -race ./...

# With coverage
cd hyper && go test -cover ./...
```

#### `make lint`
Run linters (if configured):
```bash
make lint
```

Typically runs:
- `golangci-lint run ./...`
- `eslint ui/src`

### Cleanup Targets

#### `make clean`
Remove build artifacts (preserves dependencies):
```bash
make clean
```

**What it removes:**
- `bin/hyper`
- `ui/dist/`
- `hyper/embed/ui/`

**What it preserves:**
- `node_modules/`
- Go module cache

#### `make clean-all`
Deep clean (removes all artifacts and caches):
```bash
make clean-all
```

**Warning:** Requires confirmation. Removes:
- All build artifacts
- `node_modules/`
- Go module cache

---

## Service-Specific Makefiles

For microservices projects with multiple services, create per-service Makefiles:

### Example: Multi-Service Makefile

```makefile
# Root Makefile for multi-service project

.PHONY: help build-all test-all

help:
	@echo "Multi-Service Build Commands"
	@echo "  build-all          Build all services"
	@echo "  test-all           Test all services"
	@echo "  lint-all           Lint all services"
	@echo ""
	@echo "Individual Services:"
	@echo "  make -C service-a build"
	@echo "  make -C service-b build"

build-all:
	@echo "Building all services..."
	$(MAKE) -C service-a build
	$(MAKE) -C service-b build
	$(MAKE) -C service-c build

test-all:
	@echo "Testing all services..."
	$(MAKE) -C service-a test
	$(MAKE) -C service-b test
	$(MAKE) -C service-c test

lint-all:
	@echo "Linting all services..."
	$(MAKE) -C service-a lint
	$(MAKE) -C service-b lint
	$(MAKE) -C service-c lint
```

### Example: Service-Specific Makefile

```makefile
# service-a/Makefile

SERVICE_NAME := service-a
BINARY_NAME := bin/$(SERVICE_NAME)

.PHONY: build run test lint clean

build:
	@echo "Building $(SERVICE_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/server

run: build
	@echo "Running $(SERVICE_NAME)..."
	./$(BINARY_NAME)

dev:
	air -c .air.toml

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)
```

---

## Deployment Commands (Hyperion Standard)

### Development Deployment

```bash
# Build for dev
make lint
make test
make native

# Deploy to dev namespace (if using Kubernetes)
kubectl rollout restart deployment/hyper -n dev
```

### Production Deployment

**CRITICAL:** Production deploys ONLY via CI/CD.

```yaml
# .github/workflows/deploy-prod.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build
        run: make native
      
      - name: Run tests
        run: make test
      
      - name: Deploy to production
        run: |
          # Your deployment logic
          kubectl apply -f k8s/production/
          kubectl rollout restart deployment/hyper -n prod
```

**Never run these manually in production:**
```bash
# ❌ FORBIDDEN IN PRODUCTION
kubectl rollout restart deployment/hyper -n prod
kubectl apply -f k8s/production/
```

---

## Advanced Makefile Patterns

### Pattern 1: Conditional Targets

```makefile
.PHONY: build-prod build-dev

ENV ?= dev

build-prod:
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/app ./cmd/server

build-dev:
	@echo "Building for development..."
	go build -o bin/app ./cmd/server

build:
ifeq ($(ENV),prod)
	$(MAKE) build-prod
else
	$(MAKE) build-dev
endif
```

Usage:
```bash
make build ENV=prod
make build ENV=dev
```

### Pattern 2: Service Selection

```makefile
# Build specific service
SERVICE ?= coordinator

build:
	@echo "Building service: $(SERVICE)"
	go build -o bin/$(SERVICE) ./cmd/$(SERVICE)

# Usage:
# make build SERVICE=coordinator
# make build SERVICE=worker
```

### Pattern 3: Version Tagging

```makefile
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

build:
	go build -ldflags "\
		-X main.Version=$(VERSION) \
		-X main.BuildTime=$(BUILD_TIME)" \
		-o bin/app ./cmd/server
```

### Pattern 4: Docker Integration

```makefile
DOCKER_REGISTRY := gcr.io/my-project
IMAGE_NAME := $(DOCKER_REGISTRY)/my-service
IMAGE_TAG := $(VERSION)

docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest

docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_NAME):latest

docker-run:
	docker run -p 8080:8080 $(IMAGE_NAME):latest
```

---

## Best Practices

### 1. Self-Documenting Targets
Always add descriptions after `##`:
```makefile
build: ## Build the service binary
	go build -o bin/app ./cmd/server
```

### 2. Phony Targets
Declare non-file targets as `.PHONY`:
```makefile
.PHONY: build test clean
```

### 3. Error Handling
Fail fast on errors:
```makefile
build:
	@echo "Building..."
	go build -o bin/app ./cmd/server || exit 1
```

### 4. Environment Variables
Load from `.env` file:
```makefile
include .env
export

build:
	@echo "Building with DB: $(MONGODB_URI)"
	go build -o bin/app ./cmd/server
```

### 5. Parallel Execution
Enable parallel builds:
```bash
make -j4 build-all  # Run 4 jobs in parallel
```

---

## Common Pitfalls

### 1. Tab vs Spaces
```makefile
# ❌ WRONG - Uses spaces
build:
    go build -o bin/app

# ✅ CORRECT - Uses tabs
build:
	go build -o bin/app
```

### 2. Forgetting .PHONY
```makefile
# If a file named "test" exists, this won't run
test:
	go test ./...

# ✅ CORRECT
.PHONY: test
test:
	go test ./...
```

### 3. Not Checking Command Existence
```makefile
# ❌ BAD - Fails if air not installed
dev:
	air

# ✅ GOOD - Check first
dev:
	@if ! command -v air &> /dev/null; then \
		echo "Air not installed. Run 'make install-air'"; \
		exit 1; \
	fi
	air
```

---

## Related Documentation

- [Go Microservice Scaffolding](./go-microservice-scaffolding.md) - Service structure
- [Deployment Architecture](../infrastructure/deployment-architecture.md) - CI/CD patterns
- [Configuration Reference](../configuration-reference.md) - Environment variables

---

## Troubleshooting

### Issue: "make: command not found"

**Solution:**
```bash
# macOS
brew install make

# Ubuntu/Debian
sudo apt-get install make

# Verify
make --version
```

### Issue: "No rule to make target 'build'"

**Cause:** Target doesn't exist in Makefile

**Solution:**
```bash
# List available targets
make help

# Check Makefile exists
ls -la Makefile
```

### Issue: "recipe commences before first target"

**Cause:** Using spaces instead of tabs

**Solution:**
Configure editor to use tabs for Makefiles:
```vim
# .vimrc
autocmd FileType make setlocal noexpandtab
```

### Issue: "Air not found"

**Solution:**
```bash
make install-air
# or
go install github.com/air-verse/air@latest
```
