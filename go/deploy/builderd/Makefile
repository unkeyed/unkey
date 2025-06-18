# Builderd Makefile
# Unified Makefile following Unkey service standards

# Service name and binary
SERVICE_NAME := builderd
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

# Docker configuration
DOCKER_IMAGE := builderd

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

.PHONY: build-linux
build-linux: ## Build Linux binary for deployment
	@echo "Building $(SERVICE_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/$(SERVICE_NAME)

.PHONY: quick-build
quick-build: ## Quick build without optimizations
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(SERVICE_NAME)

.PHONY: install
install: build create-user ## Install the service (requires sudo)
	@echo "Installing $(SERVICE_NAME)..."
	@sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true
	@sudo mkdir -p $(CONFIG_DIR) $(DATA_DIR) $(LOG_DIR)
	@sudo mkdir -p /opt/builderd/{scratch,rootfs,workspace,data}
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo chown -R $(SERVICE_NAME):$(SERVICE_NAME) $(DATA_DIR) $(LOG_DIR)
	@sudo chown -R $(SERVICE_NAME):$(SERVICE_NAME) /opt/builderd
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
	@echo "  - /opt/builderd"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(GEN_DIR)
	@rm -f coverage.out coverage.html

.PHONY: clean-gen
clean-gen: ## Clean generated protobuf code
	@rm -rf $(GEN_DIR)

# =============================================================================
# Code Generation
# =============================================================================

.PHONY: generate
generate: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@buf generate
	@buf lint

.PHONY: proto
proto: generate ## Alias for generate (deprecated, use generate)
	@echo "Note: 'make proto' is deprecated, use 'make generate' instead"

.PHONY: proto-lint
proto-lint: ## Lint protobuf files (alias for lint-proto)
	@buf lint

.PHONY: proto-breaking
proto-breaking: ## Check for breaking protobuf changes
	@buf breaking --against .git#branch=main

# =============================================================================
# Development
# =============================================================================

.PHONY: dev
dev: ## Run in development mode with auto-reload (requires air)
	@echo "Running $(SERVICE_NAME) in development mode..."
	@which air > /dev/null || (echo "air not found, install with: go install github.com/cosmtrek/air@latest" && exit 1)
	@air

.PHONY: run
run: build ## Build and run the service
	@echo "Running $(SERVICE_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@air

.PHONY: debug
debug: ## Run with debug logging
	@UNKEY_BUILDERD_LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: deps
deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

# =============================================================================
# Testing & Quality
# =============================================================================

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: quick-test
quick-test: ## Run tests without verbose output
	@$(GOTEST) ./...

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
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not found, install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2" && exit 1)
	$(GOLINT) run

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -w .
	@which goimports > /dev/null && goimports -w . || echo "goimports not found, install with: go install golang.org/x/tools/cmd/goimports@latest"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

# =============================================================================
# Service Management
# =============================================================================

.PHONY: service-install
service-install: install ## Install and enable service
	@echo "Service installed and enabled"

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
# CI/CD
# =============================================================================

.PHONY: ci
ci: deps generate lint vet test ## Run CI pipeline

.PHONY: release
release: clean build build-linux test ## Build release artifacts
	@echo "Release artifacts built in $(BUILD_DIR)/"

# =============================================================================
# Utilities
# =============================================================================

.PHONY: create-user
create-user: ## Create service user
	@echo "Creating $(SERVICE_NAME) user..."
	@sudo useradd -r -s /bin/false -d /opt/builderd -c "$(SERVICE_NAME) service user" $(SERVICE_NAME) 2>/dev/null || echo "User $(SERVICE_NAME) already exists"

.PHONY: env-example
env-example: ## Show example environment variables
	@echo "Example environment variables for $(SERVICE_NAME):"
	@echo "UNKEY_BUILDERD_PORT=8082"
	@echo "UNKEY_BUILDERD_OTEL_ENABLED=false"
	@echo "UNKEY_BUILDERD_STORAGE_BACKEND=local"
	@echo "UNKEY_BUILDERD_LOG_LEVEL=info"

.PHONY: version
version: ## Show version information
	@echo "$(SERVICE_NAME) version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"

.PHONY: setup
setup: deps install-tools generate ## Complete development setup

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/bufbuild/buf/cmd/buf@latest

# =============================================================================
# Docker Targets (Service-Specific)
# =============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@docker build -t $(DOCKER_IMAGE):$(VERSION) .
	@docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run: docker-build ## Build and run Docker container
	@docker run --rm -p 8082:8082 \
		-e UNKEY_BUILDERD_OTEL_ENABLED=false \
		-e UNKEY_BUILDERD_STORAGE_BACKEND=local \
		--name $(BINARY_NAME) \
		$(DOCKER_IMAGE):latest

.PHONY: docker-clean
docker-clean: ## Clean Docker images
	@docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest || true

.PHONY: docker-test-env
docker-test-env: ## Start test environment with Docker
	@echo "Starting test environment..."
	@docker run -d --name builderd-test \
		-p 8082:8082 \
		-e UNKEY_BUILDERD_OTEL_ENABLED=false \
		-e UNKEY_BUILDERD_LOG_LEVEL=debug \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(DOCKER_IMAGE):latest
	@echo "Test environment started on http://localhost:8082"
	@echo "Health check: curl http://localhost:8082/health"
	@echo "Stop with: docker stop builderd-test && docker rm builderd-test"

# =============================================================================
# Service-Specific Targets
# =============================================================================

.PHONY: example-build
example-build: build ## Run example build with Docker image
	@echo "Testing build with ghcr.io/unkeyed/unkey:f4cfee5..."
	@curl -X POST http://localhost:8082/builder.v1.BuilderService/CreateBuild \
		-H "Content-Type: application/json" \
		-H "X-Tenant-ID: example-tenant" \
		-H "X-Customer-ID: example-customer" \
		-d '{"source":{"docker_image":{"image":"ghcr.io/unkeyed/unkey:f4cfee5","tag":"f4cfee5"}},"target":{"format":"ext4","compression":"gzip"}}'