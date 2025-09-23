# Metald VM Management Service Makefile

.DEFAULT_GOAL := help

# Variables
BINARY_NAME := metald
BUILD_DIR := build
VERSION ?= 0.5.2
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := -ldflags "-s=false -w=false -X main.version=$(VERSION)"

# Colors for output
CYAN := \033[36m
RESET := \033[0m

# Targets (alphabetically ordered)
.PHONY: build build-linux check ci clean deps dev fmt help install install-bridge-8 install-bridge-32 lint release run service-logs service-logs-full service-restart service-start service-status service-stop test test-coverage test-short uninstall uninstall-bridge-systemd version vet apply-bridge-8-config apply-bridge-32-config

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/metald

build-linux: ## Build Linux binary for deployment
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/metald

check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

ci: deps lint vet test build ## Run CI pipeline locally

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

fmt: ## Format Go code
	@goimports -w .
	@gofumpt -w .

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(CYAN)<target>$(RESET)\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

install: build ## Install metald binary and systemd service
	@sudo systemctl stop metald 2>/dev/null || true
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@sudo cp contrib/systemd/metald.service /etc/systemd/system/metald.service
	@sudo systemctl daemon-reload
	@sudo systemctl start metald 2>/dev/null || true
	@echo "✓ metald installed and started"

lint: ## Run linting tools
	@which golangci-lint >/dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run --disable=godox

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

test: ## Run all tests
	@go test -json -failfast -v ./... | go run github.com/mfridman/tparse@ba2512e7be150bfcbd6f6220d517d3741f8f2f75 -all -progress -smallscreen

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
