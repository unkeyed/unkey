# AssetManagerd Makefile
# Unified Makefile following Unkey service standards

# Service name and binary
SERVICE_NAME := assetmanagerd
BINARY_NAME := $(SERVICE_NAME)

# Versioning
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y%m%d-%H%M%S)

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Directories
BUILD_DIR := build
PROTO_DIR := proto
GEN_DIR := gen
INSTALL_DIR := /usr/local/bin
SYSTEMD_DIR := /etc/systemd/system
CONFIG_DIR := /etc/$(SERVICE_NAME)
DATA_DIR := /var/lib/$(SERVICE_NAME)
LOG_DIR := /var/log/$(SERVICE_NAME)
ASSETS_DIR := /opt/vm-assets

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOLINT := golangci-lint

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =============================================================================
# Build & Install
# =============================================================================

.PHONY: all
all: clean generate build ## Clean, generate, and build

.PHONY: build
build: ## Build the binary
	@echo "Building $(SERVICE_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(SERVICE_NAME)

.PHONY: install
install: build create-user ## Install the service (requires sudo)
	@echo "Installing $(SERVICE_NAME)..."
	@sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true
	@sudo mkdir -p $(CONFIG_DIR) $(DATA_DIR) $(LOG_DIR) $(ASSETS_DIR)
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo chown $(SERVICE_NAME):$(SERVICE_NAME) $(DATA_DIR) $(ASSETS_DIR)
	@echo "Installing systemd service..."
	@sudo cp contrib/systemd/$(SERVICE_NAME).service $(SYSTEMD_DIR)/
	@sudo systemctl daemon-reload
	@sudo systemctl enable $(SERVICE_NAME)
	@sudo systemctl start $(SERVICE_NAME) 2>/dev/null || true
	@echo "$(SERVICE_NAME) installed successfully. Service started."

.PHONY: uninstall
uninstall: ## Uninstall the service (requires sudo)
	@echo "Uninstalling $(SERVICE_NAME)..."
	@sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true
	@sudo systemctl disable $(SERVICE_NAME) 2>/dev/null || true
	@sudo rm -f $(SYSTEMD_DIR)/$(SERVICE_NAME).service
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo systemctl daemon-reload
	@echo "$(SERVICE_NAME) uninstalled successfully"
	@echo "Note: Configuration and data directories preserved:"
	@echo "  - $(CONFIG_DIR)"
	@echo "  - $(DATA_DIR)"
	@echo "  - $(LOG_DIR)"
	@echo "  - $(ASSETS_DIR)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(GEN_DIR)

# =============================================================================
# Code Generation
# =============================================================================

.PHONY: generate
generate: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@buf generate

.PHONY: proto
proto: generate ## Alias for generate (deprecated, use generate)
	@echo "Note: 'make proto' is deprecated, use 'make generate' instead"

# =============================================================================
# Development
# =============================================================================

.PHONY: dev
dev: generate ## Run in development mode
	@echo "Running $(SERVICE_NAME) in development mode..."
	@go run ./cmd/$(SERVICE_NAME)

.PHONY: run
run: build ## Build and run the service
	@echo "Running $(SERVICE_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@which air > /dev/null || go install github.com/air-verse/air@latest
	@air

# =============================================================================
# Testing & Quality
# =============================================================================

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint-proto
lint-proto: ## Run protobuf linter
	@echo "Running protobuf linter..."
	@buf lint

.PHONY: lint
lint: lint-proto ## Run linter (includes protobuf linting)
	@echo "Running linter..."
	@which $(GOLINT) > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOLINT) run

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -w .

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

# =============================================================================
# Service Management
# =============================================================================

.PHONY: service-start
service-start: ## Start the service
	@sudo systemctl start $(SERVICE_NAME)
	@echo "$(SERVICE_NAME) started"

.PHONY: service-stop
service-stop: ## Stop the service
	@sudo systemctl stop $(SERVICE_NAME)
	@echo "$(SERVICE_NAME) stopped"

.PHONY: service-restart
service-restart: ## Restart the service
	@sudo systemctl restart $(SERVICE_NAME)
	@echo "$(SERVICE_NAME) restarted"

.PHONY: service-status
service-status: ## Check service status
	@sudo systemctl status $(SERVICE_NAME) --no-pager

.PHONY: service-logs
service-logs: ## Follow service logs
	@sudo journalctl -u $(SERVICE_NAME) -f

.PHONY: service-logs-tail
service-logs-tail: ## Show last 50 lines of logs
	@sudo journalctl -u $(SERVICE_NAME) -n 50 --no-pager

# =============================================================================
# Utilities
# =============================================================================

.PHONY: create-user
create-user: ## Create service user
	@echo "Creating $(SERVICE_NAME) user..."
	@sudo useradd -r -s /bin/false -d $(DATA_DIR) -c "$(SERVICE_NAME) service user" $(SERVICE_NAME) 2>/dev/null || echo "User $(SERVICE_NAME) already exists"

.PHONY: env-example
env-example: ## Show example environment variables
	@echo "Example environment variables for $(SERVICE_NAME):"
	@cat .env.example 2>/dev/null || echo "Error: .env.example not found"

.PHONY: version
version: ## Show version information
	@echo "$(SERVICE_NAME) version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"

# =============================================================================
# Service-Specific Targets
# =============================================================================

.PHONY: run-local
run-local: ## Run locally with test configuration
	UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local \
	UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/tmp/vm-assets \
	UNKEY_ASSETMANAGERD_DATABASE_PATH=/tmp/assets.db \
	UNKEY_ASSETMANAGERD_CACHE_DIR=/tmp/asset-cache \
	UNKEY_ASSETMANAGERD_OTEL_ENABLED=false \
	go run ./cmd/assetmanagerd
