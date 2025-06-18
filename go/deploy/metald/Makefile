# Metald VM Management Service Makefile

.DEFAULT_GOAL := help

# Variables
BINARY_NAME := metald
BUILD_DIR := build
VERSION ?= 0.2.0
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Colors for output
CYAN := \033[36m
RESET := \033[0m

##@ General

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(CYAN)<target>$(RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: version
version: ## Show version information
	@echo "$(BINARY_NAME) version: $(VERSION)"

##@ Development

.PHONY: deps
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

.PHONY: generate
generate: ## Generate protobuf code
	buf generate

.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

.PHONY: run
run: build ## Build and run the service
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: dev
dev: ## Run the service in development mode with auto-reload (requires air)
	@which air > /dev/null || (echo "air not found, install with: go install github.com/cosmtrek/air@latest" && exit 1)
	air

.PHONY: watch
watch: dev ## Alias for dev target

##@ Testing

.PHONY: test
test: ## Run all tests
	go test ./... -v

.PHONY: test-short
test-short: ## Run tests in short mode
	go test ./... -short

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: vm-test
vm-test: ## Run VM functionality test
	./scripts/vm-test.sh

.PHONY: multi-vm-demo
multi-vm-demo: ## Run multi-VM demo
	./scripts/multi-vm-demo.sh

##@ Code Quality

.PHONY: lint-proto
lint-proto: ## Run protobuf linter
	@echo "Running protobuf linter..."
	@buf lint

.PHONY: lint
lint: lint-proto ## Run linting tools (includes protobuf linting)
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

.PHONY: fmt
fmt: ## Format Go code
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: proto-lint
proto-lint: ## Lint protobuf files (alias for lint-proto)
	buf lint

.PHONY: proto-breaking
proto-breaking: ## Check for breaking changes in protobuf files
	buf breaking --against '.git#branch=main'

##@ Build

.PHONY: build-linux
build-linux: ## Build Linux binary for deployment
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/api

##@ Installation

# AIDEV-NOTE: The install target sets up environment configuration by copying metald.env.example
# to /etc/metald/metald.env on first install. The systemd service uses EnvironmentFile to load
# these settings, allowing easy configuration management without modifying the service file.
.PHONY: install
install: build ## Install metald binary and systemd service
	@echo "Installing metald binary and service..."
	sudo systemctl stop metald 2>/dev/null || true
	sudo mkdir -p /opt/metald
	sudo mkdir -p /var/log/metald
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	sudo cp contrib/systemd/metald.service /etc/systemd/system/metald.service
	@echo "Creating metald user and group..."
	sudo useradd -r -s /bin/false -d /opt/metald -c "Metald VM Management Service" metald 2>/dev/null || true
	sudo chown -R metald:metald /opt/metald /var/log/metald
	@echo "Creating required directories..."
	sudo mkdir -p /opt/metald/sockets
	sudo mkdir -p /opt/metald/logs
	sudo mkdir -p /srv/jailer
	sudo mkdir -p /run/netns
	sudo chown -R metald:metald /opt/metald
	sudo chown -R metald:metald /srv/jailer
	@echo "Creating tmpfiles.d configuration for network namespaces..."
	@echo "d /run/netns 0755 root root -" | sudo tee /etc/tmpfiles.d/metald-netns.conf > /dev/null
	@echo "Setting VM assets permissions..."
	sudo mkdir -p /opt/vm-assets
	sudo chown -R metald:metald /opt/vm-assets 2>/dev/null || echo "VM assets directory doesn't exist or already configured"
	@echo "Setting capabilities for integrated jailer..."
	sudo setcap 'cap_sys_admin,cap_net_admin,cap_sys_chroot,cap_setuid,cap_setgid,cap_mknod,cap_dac_override+ep' $(DESTDIR)$(BINDIR)/metald || \
		echo "WARNING: Failed to set capabilities. Metald will need to run as root."
	sudo systemctl daemon-reload
	sudo systemctl start metald 2>/dev/null || true
	@echo "Installation complete. Service started."

.PHONY: uninstall
uninstall: ## Uninstall metald service and binary
	@echo "Uninstalling metald service..."
	sudo systemctl stop metald 2>/dev/null || true
	sudo systemctl disable metald 2>/dev/null || true
	sudo rm -f /etc/systemd/system/metald.service
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	sudo rm -f /etc/sudoers.d/metald
	sudo systemctl daemon-reload
	@echo "Uninstallation complete."

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing buf..."
	@which buf > /dev/null || (curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$$(uname -s)-$$(uname -m)" -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf)

##@ Service Management

.PHONY: service-install
service-install: install ## Install and enable metald service
	@echo "Enabling metald service..."
	sudo systemctl enable metald
	@echo "Service enabled. Use 'make service-start' to start."

.PHONY: service-start
service-start: ## Start metald service
	@echo "Starting metald service..."
	sudo systemctl start metald
	@echo "Service started. Use 'make service-logs' to view logs."

.PHONY: service-stop
service-stop: ## Stop metald service
	@echo "Stopping metald service..."
	sudo systemctl stop metald

.PHONY: service-restart
service-restart: ## Restart metald service
	@echo "Restarting metald service..."
	sudo systemctl restart metald

.PHONY: service-status
service-status: ## Show metald service status
	sudo systemctl status metald

.PHONY: service-logs
service-logs: ## Follow metald service logs
	sudo journalctl -u metald -f

.PHONY: service-logs-full
service-logs-full: ## Show all metald service logs
	sudo journalctl -u metald --no-pager

##@ Development Helpers

.PHONY: debug
debug: build ## Run with debug logging
	UNKEY_METALD_OTEL_ENABLED=true ./$(BUILD_DIR)/$(BINARY_NAME)

##@ Service-Specific

.PHONY: install-sudo
install-sudo: ## Install sudoers configuration for jailer
	@echo "Installing sudoers configuration..."
	@if [ -x scripts/install-sudoers.sh ]; then \
		sudo scripts/install-sudoers.sh; \
	else \
		echo "ERROR: sudoers install script not found at scripts/install-sudoers.sh"; \
		exit 1; \
	fi

.PHONY: metrics
metrics: ## Check Prometheus metrics
	@echo "Checking Prometheus metrics..."
	curl -s http://localhost:9464/metrics | grep -E "^(vm_|process_|jailer_)" || echo "No VM metrics found"

.PHONY: health
health: ## Check service health
	@echo "Checking service health..."
	curl -s http://localhost:8080/_/health | jq . || echo "Health check failed"

##@ Utility

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

.PHONY: clean-gen
clean-gen: ## Clean generated protobuf code
	rm -rf gen/

.PHONY: setup
setup: deps install-tools generate ## Complete development setup

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

.PHONY: ci
ci: deps generate lint vet test build ## Run CI pipeline locally

.PHONY: release
release: clean ci build-linux ## Prepare release build
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux"
