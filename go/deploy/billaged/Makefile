# Billaged VM Usage Billing Service Makefile

.DEFAULT_GOAL := help

.PHONY: help build test clean lint generate run dev deps proto-lint fmt vet install uninstall service-install service-start service-stop service-restart service-status service-logs service-logs-full dev-install health stats demo quick-demo install-tools setup

# Variables
BINARY_NAME=billaged
BUILD_DIR=build

# Help target - automatically generated from target comments
help: ## Show this help message
	@echo ""
	@echo "Billaged VM Usage Billing Service - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Build targets
build: ## Build the billaged service binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/billaged

# Development targets
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

generate: ## Generate protobuf code
	buf generate
	buf lint

run: build ## Build and run the billaged service
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run the service in development mode
	go run ./cmd/billaged

# Code quality targets
test: ## Run all tests
	go test ./... -v

lint: ## Run linting tools
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

fmt: ## Format Go code
	goimports -w .

vet: ## Run go vet
	go vet ./...

# Proto targets
proto-lint: ## Lint protobuf files
	buf lint

# Clean targets
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)

# Demo targets
demo: build ## Run demo with sample data
	@echo "Starting billaged demo..."
	@echo "Will aggregate usage data every 60 seconds"
	./$(BUILD_DIR)/$(BINARY_NAME) -aggregation-interval 10s

quick-demo: build ## Run demo with 10-second aggregation
	@echo "Starting billaged quick demo..."
	@echo "Will aggregate usage data every 10 seconds"
	./$(BUILD_DIR)/$(BINARY_NAME) -port 8081 -aggregation-interval 10s

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

setup: deps install-tools generate ## Complete development setup

# Installation targets
install: build ## Install billaged binary and systemd service
	@echo "Installing billaged binary and service..."
	sudo systemctl stop billaged 2>/dev/null || true
	sudo mkdir -p /opt/billaged
	sudo mkdir -p /var/log/billaged
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	sudo cp contrib/systemd/billaged.service /etc/systemd/system/billaged.service
	@echo "Creating billaged user and group..."
	sudo useradd -r -s /bin/false -d /opt/billaged -c "Billaged VM Usage Billing Service" billaged 2>/dev/null || true
	sudo chown -R billaged:billaged /opt/billaged /var/log/billaged
	sudo systemctl daemon-reload
	sudo systemctl start billaged 2>/dev/null || true
	@echo "Installation complete. Service started."

uninstall: ## Uninstall billaged service and binary
	@echo "Uninstalling billaged service..."
	sudo systemctl stop billaged 2>/dev/null || true
	sudo systemctl disable billaged 2>/dev/null || true
	sudo rm -f /etc/systemd/system/billaged.service
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	sudo systemctl daemon-reload
	@echo "Uninstallation complete."

# Service management
service-install: install ## Install and enable billaged service
	@echo "Enabling billaged service..."
	sudo systemctl enable billaged
	@echo "Service enabled. Use 'make service-start' to start."

service-start: ## Start billaged service
	@echo "Starting billaged service..."
	sudo systemctl start billaged
	@echo "Service started. Use 'make service-logs' to view logs."

service-stop: ## Stop billaged service
	@echo "Stopping billaged service..."
	sudo systemctl stop billaged

service-restart: ## Restart billaged service
	@echo "Restarting billaged service..."
	sudo systemctl restart billaged

service-status: ## Show billaged service status
	sudo systemctl status billaged

service-logs: ## Follow billaged service logs
	sudo journalctl -u billaged -f

service-logs-full: ## Show all billaged service logs
	sudo journalctl -u billaged --no-pager

# Development service management
dev-install: build ## Install development version (hot deploy)
	@echo "Installing development version..."
	sudo systemctl stop billaged 2>/dev/null || true
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo systemctl start billaged 2>/dev/null || true
	@echo "Development version installed and started."

# Health checks
health: ## Check service health
	@echo "Checking service health..."
	curl -s http://localhost:8081/health | jq . || echo "Health check failed"

stats: ## Check service statistics
	@echo "Checking service statistics..."
	curl -s http://localhost:8081/stats | jq . || echo "Stats check failed"