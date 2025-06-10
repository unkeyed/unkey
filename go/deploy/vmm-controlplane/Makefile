# Cloud Hypervisor Control Plane Makefile

.DEFAULT_GOAL := help

.PHONY: help build test clean lint generate run dev deps docker-build docker-run build-linux test-coverage fmt vet proto-lint proto-breaking clean-gen docker-clean install-tools setup ci release quick-test quick-build debug docs o11y o11y-stop o11y-logs

# Variables
BINARY_NAME=chcp-api
BUILD_DIR=build
DOCKER_IMAGE=cloud-hypervisor-controlplane
VERSION?=dev

# Help target - automatically generated from target comments
help: ## Show this help message
	@echo ""
	@echo "Cloud Hypervisor Control Plane - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Build targets
build: ## Build the API server binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

build-linux: ## Build Linux binary for deployment
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/api

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
	go fmt ./...

vet: ## Run go vet
	go vet ./...

# Proto targets
proto-lint: ## Lint protobuf files
	buf lint

proto-breaking: ## Check for breaking changes in protobuf files
	buf breaking --against '.git#branch=main'

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
		-e CHCP_CH_ENDPOINT=http://host.docker.internal:8080 \
		$(DOCKER_IMAGE):latest

docker-clean: ## Clean Docker images
	docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest 2>/dev/null || true

# Development setup targets
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing buf..."
	@which buf > /dev/null || (curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$$(uname -s)-$$(uname -m)" -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf)

setup: deps install-tools generate ## Complete development setup

# CI/CD targets
ci: deps generate lint test build ## Run CI pipeline locally

# Release targets
release: clean ci build-linux ## Prepare release build
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux"

# Environment setup
.env: ## Create example environment file
	@echo "Creating .env file with example configuration..."
	@echo "# Cloud Hypervisor Control Plane Configuration" > .env
	@echo "CHCP_PORT=8080" >> .env
	@echo "CHCP_ADDRESS=0.0.0.0" >> .env
	@echo "CHCP_BACKEND=cloudhypervisor" >> .env
	@echo "CHCP_CH_ENDPOINT=http://localhost:8080" >> .env
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
	CHCP_LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME)

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

# TODO: Replace with proper unit tests for OTEL config and integration tests
# test-otel: build ## Test OpenTelemetry integration with different sampling rates
# 	@echo "Testing OpenTelemetry integration..."
# 	@./test-otel.sh

# Documentation targets
docs: ## Generate API documentation (placeholder)
	@echo "API documentation generation not yet implemented"
	@echo "ConnectRPC services available at: /cloudhypervisor.v1.VmService/*"
	@echo "Health check available at: /health"
