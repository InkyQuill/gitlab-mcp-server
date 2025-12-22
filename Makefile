.PHONY: setup build clean test docker-build install-mcp help

# Variables
BINARY_NAME=gitlab-mcp-server
BINARY_PATH=bin/$(BINARY_NAME)
INSTALLER_BINARY=bin/install
GO_VERSION_MIN=1.23

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install prerequisites and dependencies
	@echo "Checking prerequisites..."
	@command -v go >/dev/null 2>&1 || { echo "Error: Go is not installed. Please install Go $(GO_VERSION_MIN) or later."; exit 1; }
	@echo "Go found: $$(go version)"
	@go version | awk -v min=$(GO_VERSION_MIN) '{if ($$3 < "go" min) {print "Error: Go version must be $(GO_VERSION_MIN) or later"; exit 1}}' || { echo "Error: Go version must be $(GO_VERSION_MIN) or later"; exit 1; }
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Prerequisites installed successfully!"

build: setup ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@go build -o $(BINARY_PATH) ./cmd/gitlab-mcp-server
	@echo "Build complete: $(BINARY_PATH)"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	@go test ./...

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t gitlab-mcp-server:latest .
	@echo "Docker image built: gitlab-mcp-server:latest"

install-mcp: build ## Build installer and run MCP configuration
	@echo "Building installer..."
	@mkdir -p bin
	@go build -o $(INSTALLER_BINARY) ./cmd/install
	@echo "Running installer..."
	@$(INSTALLER_BINARY)

