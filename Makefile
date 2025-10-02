.PHONY: help build run-mcp run-web run-all test clean install configure-claude test-connection

# Load environment variables from .env file
include .env
export

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the MCP server binary
	@echo "Building MCP server..."
	cd coordinator/mcp-server && go build -o hyperion-coordinator-mcp main.go
	@echo "✓ Build complete: coordinator/mcp-server/hyperion-coordinator-mcp"

install: ## Install all dependencies (Go + Node)
	@echo "Installing Go dependencies..."
	cd coordinator/mcp-server && go mod download
	@echo "✓ Go dependencies installed"
	@echo "Installing Node.js dependencies..."
	cd coordinator/ui && npm install
	@echo "✓ Node.js dependencies installed"
	@echo "✓ All dependencies installed"

run-mcp: ## Run the MCP server (stdio mode for Claude Code)
	@echo "Starting MCP server in stdio mode..."
	cd coordinator/mcp-server && TRANSPORT_MODE=stdio ./hyperion-coordinator-mcp

run-mcp-http: ## Run the MCP server in HTTP mode on port 7778
	@echo "Starting MCP server in HTTP mode on port $(MCP_PORT)..."
	@echo "HTTP endpoint: http://localhost:$(MCP_PORT)/mcp"
	cd coordinator/mcp-server && TRANSPORT_MODE=http MCP_PORT=$(MCP_PORT) ./hyperion-coordinator-mcp

run-web: ## Run the web UI on port 7777
	@echo "Starting web UI on port $(WEB_PORT)..."
	cd coordinator/ui && npm run dev -- --port $(WEB_PORT)

run-all: ## Run both MCP server (HTTP) and web UI in parallel
	@echo "Starting dev-squad system..."
	@echo "MCP Server: http://localhost:$(MCP_PORT)/mcp"
	@echo "Web UI: http://localhost:$(WEB_PORT)"
	@make -j2 run-mcp-http run-web

test: ## Run tests
	@echo "Running MCP server tests..."
	cd coordinator/mcp-server && go test ./...
	@echo "✓ All tests passed"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f coordinator/mcp-server/hyperion-coordinator-mcp
	rm -f coordinator/mcp-server/*.tgz
	@echo "✓ Clean complete"

configure-claude: ## Add MCP server to Claude Code (stdio mode)
	@echo "Configuring Claude Code MCP server..."
	cd coordinator/mcp-server && ./add-to-claude-code.sh
	@echo "✓ Claude Code configured"

configure-claude-http: ## Add MCP server to Claude Code (HTTP mode using docker-compose ports)
	@echo "Configuring Claude Code for HTTP mode..."
	@echo "Adding HTTP transport configuration to Claude Code settings..."
	@mkdir -p ~/.config/claude
	@echo '{' > ~/.config/claude/mcp-http-config.json
	@echo '  "mcpServers": {' >> ~/.config/claude/mcp-http-config.json
	@echo '    "hyperion-coordinator": {' >> ~/.config/claude/mcp-http-config.json
	@echo '      "url": "http://localhost:7778/mcp",' >> ~/.config/claude/mcp-http-config.json
	@echo '      "transport": "http"' >> ~/.config/claude/mcp-http-config.json
	@echo '    }' >> ~/.config/claude/mcp-http-config.json
	@echo '  }' >> ~/.config/claude/mcp-http-config.json
	@echo '}' >> ~/.config/claude/mcp-http-config.json
	@echo ""
	@echo "✓ HTTP configuration created at ~/.config/claude/mcp-http-config.json"
	@echo ""
	@echo "Add this to your Claude Code settings:"
	@echo "  Port: 7778 (maps to container port 8095)"
	@echo "  URL: http://localhost:7778/mcp"
	@echo ""
	@cat ~/.config/claude/mcp-http-config.json
	@echo ""
	@echo "Note: Make sure docker-compose is running first!"

test-connection: ## Test MongoDB and Qdrant connections
	@echo "Testing connections..."
	cd coordinator/mcp-server && ./test-connection.sh
	@echo "✓ Connection test complete"
