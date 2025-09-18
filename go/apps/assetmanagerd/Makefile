# AssetManagerd Makefile
# Unified Makefile following Unkey service standards

# Service name and binary
SERVICE_NAME := assetmanagerd
BINARY_NAME := $(SERVICE_NAME)

# Versioning
VERSION ?= 0.5.1
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
GOFMT := goimports
GOLINT := golangci-lint

# Default target
.DEFAULT_GOAL := help

# Targets (alphabetically ordered)
.PHONY: all build check clean deps dev env-example fmt generate help install lint lint-proto run service-logs service-logs-tail service-restart service-start service-status service-stop setup test test-coverage uninstall version vet

all: clean build ## Clean, generate, and build

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(SERVICE_NAME)

check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)
	@rm -rf $(GEN_DIR)

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

dev: generate ## Run in development mode
	@go run ./cmd/$(SERVICE_NAME)

fmt: ## Format code
	@$(GOFMT) -w .

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: build ## Install the service (requires sudo)
	@sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true
	@sudo mkdir -p $(CONFIG_DIR)
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo cp contrib/systemd/$(SERVICE_NAME).service $(SYSTEMD_DIR)/
	@sudo systemctl daemon-reload
	@sudo systemctl enable $(SERVICE_NAME) >/dev/null 2>&1
	@sudo systemctl start $(SERVICE_NAME) 2>/dev/null || true
	@echo "✓ $(SERVICE_NAME) installed and started"

lint: lint-proto ## Run linter (includes protobuf linting)
	@which $(GOLINT) >/dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GOLINT) run

lint-proto: ## Run protobuf linter
	@buf lint

run: build ## Build and run the service
	@./$(BUILD_DIR)/$(BINARY_NAME)

service-logs: ## Follow service logs
	@sudo journalctl -u $(SERVICE_NAME) -f

service-logs-tail: ## Show last 50 lines of logs
	@sudo journalctl -u $(SERVICE_NAME) -n 50 --no-pager

service-restart: ## Restart the service
	@sudo systemctl restart $(SERVICE_NAME)
	@echo "✓ $(SERVICE_NAME) restarted"

service-start: ## Start the service
	@sudo systemctl start $(SERVICE_NAME)
	@echo "✓ $(SERVICE_NAME) started"

service-status: ## Check service status
	@sudo systemctl status $(SERVICE_NAME) --no-pager

service-stop: ## Stop the service
	@sudo systemctl stop $(SERVICE_NAME)
	@echo "✓ $(SERVICE_NAME) stopped"

setup: deps generate ## Complete development setup

test: ## Run tests
	@$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

uninstall: ## Uninstall the service (requires sudo)
	@sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true
	@sudo systemctl disable $(SERVICE_NAME) 2>/dev/null || true
	@sudo rm -f $(SYSTEMD_DIR)/$(SERVICE_NAME).service
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo systemctl daemon-reload
	@echo "✓ $(SERVICE_NAME) uninstalled (config/data preserved)"

version: ## Show version information
	@echo "$(SERVICE_NAME) version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"

vet: ## Run go vet
	@$(GOVET) ./...

