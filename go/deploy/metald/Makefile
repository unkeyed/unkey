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
.PHONY: build build-linux check ci clean deps dev fmt help install install-bridge-8 install-bridge-32 lint release run service-logs service-logs-full service-restart service-start service-status service-stop test test-coverage test-short uninstall uninstall-bridge-systemd version vet apply-bridge-8-config apply-bridge-32-config

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

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

dev: ## Run the service in development mode
	@go run ./cmd/metald

fmt: ## Format Go code
	@goimports -w .

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(CYAN)<target>$(RESET)\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

install: build ## Install metald binary and systemd service
	@sudo systemctl stop metald 2>/dev/null || true
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@sudo cp contrib/systemd/metald.service /etc/systemd/system/metald.service
	@echo "d /run/netns 0755 root root -" | sudo tee /etc/tmpfiles.d/metald-netns.conf >/dev/null
	@echo "→ Checking for existing bridge configuration..."
	@if [ -f /etc/systemd/system/metald-bridge-8.service ]; then \
		echo "→ Reapplying 8-bridge configuration"; \
		$(MAKE) apply-bridge-8-config; \
	elif [ -f /etc/systemd/system/metald-bridge-32.service ]; then \
		echo "→ Reapplying 32-bridge configuration"; \
		$(MAKE) apply-bridge-32-config; \
	else \
		echo "→ No multi-bridge configuration found"; \
	fi
	@sudo systemctl daemon-reload
	@sudo systemctl start metald 2>/dev/null || true
	@echo "✓ metald installed and started"

install-bridge-8: ## Install 8-bridge multi-tenant configuration (with cleanup of other configs)
	@echo "Installing 8-bridge multi-tenant configuration..."
	@echo "⚠ WARNING: This will remove any existing bridge configuration"
	@$(MAKE) uninstall-bridge-systemd >/dev/null 2>&1 || true
	@echo "→ Installing 8-bridge systemd-networkd configuration (separate files)"
	@sudo cp contrib/systemd/multi-bridge-8-fixed/*.netdev /etc/systemd/network/
	@sudo cp contrib/systemd/multi-bridge-8-fixed/*.network /etc/systemd/network/
	@echo "→ Installing metald-bridge-8 systemd service"
	@sudo cp contrib/systemd/metald-bridge-8.service /etc/systemd/system/metald-bridge-8.service
	@echo "→ Applying service configuration to metald.service"
	@$(MAKE) apply-bridge-8-config
	@echo "→ Enabling systemd-networkd (if not already enabled)"
	@sudo systemctl enable systemd-networkd 2>/dev/null || true
	@echo "→ Enabling metald-bridge-8 service"
	@sudo systemctl enable metald-bridge-8.service
	@echo "→ Reloading systemd configuration"
	@sudo systemctl daemon-reload
	@echo "→ Starting 8-bridge infrastructure"
	@sudo systemctl start metald-bridge-8.service
	@echo "→ Verifying bridge configuration (checking first few bridges)"
	@sleep 3
	@for i in 0 1 2; do \
		ip link show br-tenant-$$i >/dev/null 2>&1 && echo "✓ Bridge br-tenant-$$i exists" || echo "✗ Bridge br-tenant-$$i missing"; \
	done
	@echo "✅ 8-bridge infrastructure installed"

install-bridge-32: ## Install 32-bridge multi-tenant configuration (with cleanup of other configs)
	@echo "Installing 32-bridge multi-tenant configuration..."
	@echo "⚠ WARNING: This will remove any existing bridge configuration"
	@$(MAKE) uninstall-bridge-systemd >/dev/null 2>&1 || true
	@echo "→ Installing 32-bridge systemd-networkd configuration (separate files)"
	@sudo cp contrib/systemd/multi-bridge-32-fixed/*.netdev /etc/systemd/network/
	@sudo cp contrib/systemd/multi-bridge-32-fixed/*.network /etc/systemd/network/
	@echo "→ Installing metald-bridge-32 systemd service"
	@sudo cp contrib/systemd/metald-bridge-32.service /etc/systemd/system/metald-bridge-32.service
	@echo "→ Applying service configuration to metald.service"
	@$(MAKE) apply-bridge-32-config
	@echo "→ Enabling systemd-networkd (if not already enabled)"
	@sudo systemctl enable systemd-networkd 2>/dev/null || true
	@echo "→ Enabling metald-bridge-32 service"
	@sudo systemctl enable metald-bridge-32.service
	@echo "→ Reloading systemd configuration"
	@sudo systemctl daemon-reload
	@echo "→ Starting 32-bridge infrastructure"
	@sudo systemctl start metald-bridge-32.service
	@echo "→ Verifying bridge configuration (checking first few bridges)"
	@sleep 3
	@for i in 0 1 2; do \
		ip link show br-tenant-$$i >/dev/null 2>&1 && echo "✓ Bridge br-tenant-$$i exists" || echo "✗ Bridge br-tenant-$$i missing"; \
	done
	@echo "✅ 32-bridge infrastructure installed"

apply-bridge-8-config: ## Apply 8-bridge configuration to metald service (internal helper)
	@echo "→ Setting bridge count environment in metald service"
	@sudo sed -i '/^Environment=UNKEY_METALD_NETWORK_BRIDGE_COUNT=/d' /etc/systemd/system/metald.service 2>/dev/null || true
	@sudo sed -i '/^# TLS\/SPIFFE configuration (REQUIRED)/i Environment=UNKEY_METALD_NETWORK_BRIDGE_COUNT=8' /etc/systemd/system/metald.service
	@echo "→ Updating metald service dependencies to use metald-bridge-8"
	@sudo sed -i 's/metald-bridge-32\.service/metald-bridge-8.service/g' /etc/systemd/system/metald.service
	@sudo sed -i 's/metald-bridge\.service/metald-bridge-8.service/g' /etc/systemd/system/metald.service

apply-bridge-32-config: ## Apply 32-bridge configuration to metald service (internal helper)
	@echo "→ Setting bridge count environment in metald service"
	@sudo sed -i '/^Environment=UNKEY_METALD_NETWORK_BRIDGE_COUNT=/d' /etc/systemd/system/metald.service 2>/dev/null || true
	@sudo sed -i '/^# TLS\/SPIFFE configuration (REQUIRED)/i Environment=UNKEY_METALD_NETWORK_BRIDGE_COUNT=32' /etc/systemd/system/metald.service
	@echo "→ Updating metald service dependencies to use metald-bridge-32"
	@sudo sed -i 's/metald-bridge-8\.service/metald-bridge-32.service/g' /etc/systemd/system/metald.service
	@sudo sed -i 's/metald-bridge\.service/metald-bridge-32.service/g' /etc/systemd/system/metald.service

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

uninstall-bridge-systemd: ## Uninstall VM network bridge infrastructure (all configurations)
	@echo "Uninstalling VM network bridge infrastructure..."
	@echo "⚠ WARNING: This will remove all bridge configurations and may disrupt running VMs"
	@echo "→ Stopping metald service first"
	@sudo systemctl stop metald 2>/dev/null || true
	@echo "→ Stopping and disabling all bridge services"
	@sudo systemctl stop metald-bridge.service 2>/dev/null || true
	@sudo systemctl stop metald-bridge-8.service 2>/dev/null || true
	@sudo systemctl stop metald-bridge-32.service 2>/dev/null || true
	@sudo systemctl disable metald-bridge.service 2>/dev/null || true
	@sudo systemctl disable metald-bridge-8.service 2>/dev/null || true
	@sudo systemctl disable metald-bridge-32.service 2>/dev/null || true
	@echo "→ Removing bridge infrastructure"
	@for i in $$(seq 0 31); do \
		sudo ip link set br-tenant-$$i down 2>/dev/null || true; \
		sudo ip link del br-tenant-$$i 2>/dev/null || true; \
	done
	@echo "→ Removing systemd service files"
	@sudo rm -f /etc/systemd/system/metald-bridge.service
	@sudo rm -f /etc/systemd/system/metald-bridge-8.service
	@sudo rm -f /etc/systemd/system/metald-bridge-32.service
	@echo "→ Removing systemd-networkd configuration"
	@sudo rm -f /etc/systemd/network/10-br-tenant-*.netdev
	@sudo rm -f /etc/systemd/network/10-br-tenant-*.network
	@echo "→ Removing bridge count environment from metald service"
	@sudo sed -i '/^Environment=UNKEY_METALD_NETWORK_BRIDGE_COUNT=/d' /etc/systemd/system/metald.service 2>/dev/null || true
	@echo "→ Reloading systemd configuration"
	@sudo systemctl daemon-reload
	@echo "→ Restarting systemd-networkd to apply changes"
	@sudo systemctl restart systemd-networkd 2>/dev/null || true
	@echo "✅ VM network bridge infrastructure uninstalled"

version: ## Show version information
	@echo "$(BINARY_NAME) version: $(VERSION)"

vet: ## Run go vet
	@go vet ./...

