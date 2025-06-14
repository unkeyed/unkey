# Builderd Multi-Tenant Build Service Makefile

.DEFAULT_GOAL := help

.PHONY: help build test clean lint generate run dev deps docker-build docker-run build-linux test-coverage fmt vet proto-lint proto-breaking clean-gen docker-clean install-tools setup ci release quick-test quick-build debug docs install uninstall service-install service-start service-stop service-logs service-status

# Variables
BINARY_NAME=builderd
BUILD_DIR=build
DOCKER_IMAGE=builderd
VERSION?=0.1.0

# Help target - automatically generated from target comments
help: ## Show this help message
	@echo ""
	@echo "Builderd Multi-Tenant Build Service - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Build targets
build: ## Build the builderd service binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/builderd

build-linux: ## Build Linux binary for deployment
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/builderd

quick-build: ## Quick build without optimizations
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/builderd

# Development targets
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

generate: ## Generate protobuf code
	buf generate
	buf lint

run: build ## Build and run the builderd service
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run the server in development mode with auto-reload (requires air)
	@which air > /dev/null || (echo "air not found, install with: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# Testing targets
test: ## Run all tests
	go test -v ./...

quick-test: ## Run tests without verbose output
	go test ./...

test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality targets
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2" && exit 1)
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not found, install with: go install golang.org/x/tools/cmd/goimports@latest"

vet: ## Run go vet
	go vet ./...

# Proto targets
proto-lint: ## Lint protobuf files
	buf lint

proto-breaking: ## Check for breaking protobuf changes
	buf breaking --against .git#branch=main

# Docker targets
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run: docker-build ## Build and run Docker container
	docker run --rm -p 8082:8082 \
		-e UNKEY_BUILDERD_OTEL_ENABLED=false \
		-e UNKEY_BUILDERD_STORAGE_BACKEND=local \
		--name $(BINARY_NAME) \
		$(DOCKER_IMAGE):latest

docker-clean: ## Clean Docker images
	docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest || true

# Installation targets (requires root/sudo)
install: build ## Install builderd system-wide
	@echo "Installing $(BINARY_NAME)..."
	sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Creating builderd user and directories..."
	sudo useradd -r -s /bin/false -d /opt/builderd builderd 2>/dev/null || true
	sudo mkdir -p /opt/builderd/{scratch,rootfs,workspace,data}
	sudo mkdir -p /var/log/builderd
	sudo chown -R builderd:builderd /opt/builderd
	sudo chown -R builderd:builderd /var/log/builderd
	@echo "Installing systemd service..."
	sudo cp contrib/systemd/builderd.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Installation complete. Use 'make service-start' to start the service."

uninstall: ## Uninstall builderd from system
	@echo "Stopping and disabling service..."
	sudo systemctl stop builderd || true
	sudo systemctl disable builderd || true
	@echo "Removing files..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	sudo rm -f /etc/systemd/system/builderd.service
	sudo systemctl daemon-reload
	@echo "Note: User 'builderd' and data directories were preserved."
	@echo "To remove completely: sudo deluser builderd && sudo rm -rf /opt/builderd /var/log/builderd"

# Service management targets
service-install: ## Enable builderd systemd service
	sudo systemctl enable builderd

service-start: ## Start builderd service
	sudo systemctl start builderd

service-stop: ## Stop builderd service
	sudo systemctl stop builderd

service-restart: ## Restart builderd service
	sudo systemctl restart builderd

service-status: ## Show builderd service status
	sudo systemctl status builderd

service-logs: ## Show builderd service logs
	sudo journalctl -u builderd -f

# Setup and development tools
install-tools: ## Install development tools
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

setup: deps install-tools generate ## Complete development setup

# CI/CD targets
ci: deps generate lint vet test ## Run CI pipeline

release: clean build build-linux test ## Build release artifacts
	@echo "Release artifacts built in $(BUILD_DIR)/"

# Cleanup targets
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

clean-gen: ## Clean generated files
	rm -rf gen/

# Debug and testing targets
debug: ## Run builderd with debug logging
	UNKEY_BUILDERD_LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME)

# Docker test environment
docker-test-env: ## Start test environment with Docker
	@echo "Starting test environment..."
	docker run -d --name builderd-test \
		-p 8082:8082 \
		-e UNKEY_BUILDERD_OTEL_ENABLED=false \
		-e UNKEY_BUILDERD_LOG_LEVEL=debug \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(DOCKER_IMAGE):latest
	@echo "Test environment started on http://localhost:8082"
	@echo "Health check: curl http://localhost:8082/health"
	@echo "Stop with: docker stop builderd-test && docker rm builderd-test"

# Examples and demonstrations
example-build: build ## Run example build with Docker image
	@echo "Testing build with ghcr.io/unkeyed/unkey:f4cfee5..."
	curl -X POST http://localhost:8082/builder.v1.BuilderService/CreateBuild \
		-H "Content-Type: application/json" \
		-H "X-Tenant-ID: example-tenant" \
		-H "X-Customer-ID: example-customer" \
		-d '{"source":{"docker_image":{"image":"ghcr.io/unkeyed/unkey:f4cfee5","tag":"f4cfee5"}},"target":{"format":"ext4","compression":"gzip"}}'
