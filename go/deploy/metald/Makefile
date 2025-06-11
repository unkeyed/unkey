# Metald VM Management Service Makefile

.DEFAULT_GOAL := help

.PHONY: help build test clean lint generate run dev deps docker-build docker-run build-linux test-coverage fmt vet proto-lint proto-breaking clean-gen docker-clean install-tools setup ci release quick-test quick-build debug docs o11y o11y-stop o11y-logs install uninstall service-install service-start service-stop service-logs stress-test vm-test multi-vm-demo quick-stress-test

# Variables
BINARY_NAME=metald
BUILD_DIR=build
DOCKER_IMAGE=metald
VERSION?=dev

# Help target - automatically generated from target comments
help: ## Show this help message
	@echo ""
	@echo "Metald VM Management Service - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Build targets
build: ## Build the API server binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

build-linux: ## Build Linux binary for deployment
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/api

stress-test: ## Build stress test binary
	@echo "Building stress test..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w" -o $(BUILD_DIR)/stress-test ./cmd/stress-test

# Development targets
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

generate: ## Generate protobuf code
	buf generate
	buf lint

run: build ## Build and run the API server
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run the server in development mode with auto-reload (requires air)
	@which air > /dev/null || (echo "air not found, install with: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# Code quality targets
test: ## Run all tests
	go test ./... -v

test-coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linting tools
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

fmt: ## Format Go code
	goimports -w .

vet: ## Run go vet
	go vet ./...

# Testing targets
vm-test: ## Run VM functionality test
	./scripts/vm-test.sh

multi-vm-demo: ## Run multi-VM demo
	./scripts/multi-vm-demo.sh

quick-stress-test: ## Run quick stress test
	./scripts/quick-stress-test.sh

# Proto targets
proto-lint: ## Lint protobuf files
	buf lint

proto-breaking: ## Check for breaking changes in protobuf files
	buf breaking --against '.git#branch=main'

# Installation targets
install: build ## Install metald binary and systemd service
	@echo "Installing metald binary and service..."
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
	sudo chown -R metald:metald /opt/metald
	sudo chown -R metald:metald /srv/jailer
	@echo "Setting VM assets permissions..."
	sudo mkdir -p /opt/vm-assets
	sudo chown -R metald:metald /opt/vm-assets 2>/dev/null || echo "VM assets directory doesn't exist or already configured"
	sudo systemctl daemon-reload
	@echo "Installation complete. Use 'make service-start' to start the service."

uninstall: ## Uninstall metald service and binary
	@echo "Uninstalling metald service..."
	sudo systemctl stop metald 2>/dev/null || true
	sudo systemctl disable metald 2>/dev/null || true
	sudo rm -f /etc/systemd/system/metald.service
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	sudo systemctl daemon-reload
	@echo "Uninstallation complete."

# Service management
service-install: install ## Install and enable metald service
	@echo "Enabling metald service..."
	sudo systemctl enable metald
	@echo "Service enabled. Use 'make service-start' to start."

service-start: ## Start metald service
	@echo "Starting metald service..."
	sudo systemctl start metald
	@echo "Service started. Use 'make service-logs' to view logs."

service-stop: ## Stop metald service
	@echo "Stopping metald service..."
	sudo systemctl stop metald

service-restart: ## Restart metald service
	@echo "Restarting metald service..."
	sudo systemctl restart metald

service-status: ## Show metald service status
	sudo systemctl status metald

service-logs: ## Follow metald service logs
	sudo journalctl -u metald -f

service-logs-full: ## Show all metald service logs
	sudo journalctl -u metald --no-pager

# Development service management
dev-install: build ## Install development version (hot deploy)
	@echo "Installing development version..."
	sudo systemctl stop metald 2>/dev/null || true
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo systemctl start metald 2>/dev/null || true
	@echo "Development version installed and started."

dev-logs: ## Show recent development logs
	sudo journalctl -u metald -f --since "1 minute ago"

# Production deployment with jailer
prod-install: build ## Install production version with jailer support
	@echo "Installing production version with jailer support..."
	sudo systemctl stop metald 2>/dev/null || true
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Updating service configuration for production..."
	sudo sed -i 's/UNKEY_METALD_JAILER_ENABLED=false/UNKEY_METALD_JAILER_ENABLED=true/' /etc/systemd/system/metald.service
	sudo systemctl daemon-reload
	sudo systemctl start metald
	@echo "Production version with jailer installed and started."

# Monitoring and metrics
metrics: ## Check Prometheus metrics
	@echo "Checking Prometheus metrics..."
	curl -s http://localhost:9464/metrics | grep -E "^(vm_|process_|jailer_)" || echo "No VM metrics found"

health: ## Check service health
	@echo "Checking service health..."
	curl -s http://localhost:8080/_/health | jq . || echo "Health check failed"

# Clean targets
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

clean-gen: ## Clean generated protobuf code
	rm -rf gen/

# Docker targets
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	docker run --rm -p 8080:8080 \
		-e UNKEY_METALD_BACKEND=firecracker \
		-e UNKEY_METALD_OTEL_ENABLED=true \
		$(DOCKER_IMAGE):latest

docker-clean: ## Clean Docker images
	docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest 2>/dev/null || true

# Development setup targets
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing buf..."
	@which buf > /dev/null || (curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$$(uname -s)-$$(uname -m)" -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf)

setup: deps install-tools generate ## Complete development setup

# CI/CD targets
ci: deps generate lint vet test build ## Run CI pipeline locally

# Release targets
release: clean ci build-linux ## Prepare release build
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux"

# Environment setup
.env: ## Create example environment file
	@echo "Creating .env file with example configuration..."
	@echo "# Metald VM Management Service Configuration" > .env
	@echo "UNKEY_METALD_PORT=8080" >> .env
	@echo "UNKEY_METALD_ADDRESS=0.0.0.0" >> .env
	@echo "UNKEY_METALD_BACKEND=firecracker" >> .env
	@echo "UNKEY_METALD_OTEL_ENABLED=true" >> .env
	@echo "UNKEY_METALD_JAILER_ENABLED=false" >> .env
	@echo ".env file created with default values"

# Quick development targets
quick-test: ## Run tests without dependencies check
	go test ./... -short

quick-build: ## Quick build without cleaning
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

test-e2e: ## Run end-to-end tests (placeholder)
	@echo "E2E tests not yet implemented"

# Debugging targets
debug: ## Run with debug logging
	UNKEY_METALD_OTEL_ENABLED=true ./$(BUILD_DIR)/$(BINARY_NAME)

# Observability targets
o11y: ## Start observability stack (Grafana LGTM) with Podman
	@echo "Starting observability stack with Podman..."
	@podman run -d --name otel-lgtm \
		-p 0.0.0.0:3000:3000 \
		-p 0.0.0.0:4317:4317 \
		-p 0.0.0.0:4318:4318 \
		-e OTEL_METRIC_EXPORT_INTERVAL=10000 \
		grafana/otel-lgtm:latest
	@echo "Observability stack started:"
	@echo "  Grafana UI:     http://0.0.0.0:3000 (admin/admin)"
	@echo "  OTLP gRPC:      0.0.0.0:4317"
	@echo "  OTLP HTTP:      0.0.0.0:4318"
	@echo ""
	@echo "To stop: make o11y-stop"

o11y-stop: ## Stop observability stack
	@echo "Stopping observability stack..."
	@podman stop otel-lgtm 2>/dev/null || true
	@podman rm otel-lgtm 2>/dev/null || true
	@echo "Observability stack stopped"

o11y-logs: ## Show observability stack logs
	@podman logs -f otel-lgtm

# Documentation targets
docs: ## Generate API documentation (placeholder)
	@echo "API documentation generation not yet implemented"
	@echo "ConnectRPC services available at: /vmprovisioner.v1.VmService/*"
	@echo "Health check available at: /_/health"
