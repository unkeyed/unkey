# Metald VM Management Service Makefile

.DEFAULT_GOAL := help

# Variables
BINARY_NAME := metald
BUILD_DIR := build
VERSION ?= 0.5.2
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Colors for output
CYAN := \033[36m
RESET := \033[0m

# Targets (alphabetically ordered)
.PHONY: build build-linux check ci clean debug deps dev fmt health help install install-tools lint metrics release run service-logs service-logs-full service-restart service-start service-status service-stop setup test test-coverage test-short uninstall version vet

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/metald

build-linux: ## Build Linux binary for deployment
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/metald

check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

ci: deps lint vet test build ## Run CI pipeline locally

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html


debug: build ## Run with debug logging
	@UNKEY_METALD_OTEL_ENABLED=true ./$(BUILD_DIR)/$(BINARY_NAME)

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

dev: ## Run the service in development mode
	@go run ./cmd/metald

fmt: ## Format Go code
	@goimports -w .


health: ## Check service health
	@curl -s http://localhost:8080/_/health | jq . || echo "Health check failed"

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(CYAN)<target>$(RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# AIDEV-NOTE: The install target sets up environment configuration by copying metald.env.example
# to /etc/metald/metald.env on first install. The systemd service uses EnvironmentFile to load
# these settings, allowing easy configuration management without modifying the service file.
install: build ## Install metald binary and systemd service
	@sudo systemctl stop metald 2>/dev/null || true
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@sudo cp contrib/systemd/metald.service /etc/systemd/system/metald.service
	@echo "d /run/netns 0755 root root -" | sudo tee /etc/tmpfiles.d/metald-netns.conf >/dev/null
	@sudo systemctl daemon-reload
	@sudo systemctl start metald 2>/dev/null || true
	@echo "✓ metald installed and started"


lint: ## Run linting tools
	@which golangci-lint >/dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run --disable=godox

metrics: ## Check Prometheus metrics
	@curl -s http://localhost:9464/metrics | grep -E "^(vm_|process_|jailer_)" || echo "No VM metrics found"


release: clean ci build-linux ## Prepare release build
	@echo "✓ Release build: $(BUILD_DIR)/$(BINARY_NAME)-linux"

run: build ## Build and run the service
	@./$(BUILD_DIR)/$(BINARY_NAME)

service-logs: ## Follow metald service logs
	@sudo journalctl -u metald -f

service-logs-full: ## Show all metald service logs
	@sudo journalctl -u metald --no-pager

service-restart: ## Restart metald service
	@sudo systemctl restart metald
	@echo "✓ metald restarted"

service-start: ## Start metald service
	@sudo systemctl start metald
	@echo "✓ metald started"

service-status: ## Show metald service status
	@sudo systemctl status metald

service-stop: ## Stop metald service
	@sudo systemctl stop metald
	@echo "✓ metald stopped"

setup: deps ## Complete development setup

test: ## Run all tests
	@go test ./... -v

test-coverage: ## Run tests with coverage report
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

test-short: ## Run tests in short mode
	@go test ./... -short

uninstall: ## Uninstall metald service and binary
	@sudo systemctl stop metald 2>/dev/null || true
	@sudo systemctl disable metald 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/metald.service
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo rm -f /etc/sudoers.d/metald
	@sudo systemctl daemon-reload
	@echo "✓ metald uninstalled"

version: ## Show version information
	@echo "$(BINARY_NAME) version: $(VERSION)"

vet: ## Run go vet
	@go vet ./...