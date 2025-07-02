# Billaged VM Usage Billing Service Makefile

.DEFAULT_GOAL := help

# Variables
BINARY_NAME=billaged
BUILD_DIR=build
VERSION ?= 0.5.1
COMMIT=$(shell git rev-parse --short HEAD || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Targets (alphabetically ordered)
.PHONY: build check clean clean-gen create-user deps dev env-example fmt generate health help install lint lint-proto run service-logs service-logs-tail service-restart service-start service-status service-stop setup test test-coverage uninstall version vet

build: generate deps ## Build the billaged service binary
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/billaged

check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

clean-gen: ## Clean generated protobuf code
	rm -rf ./gen

create-user: ## Create service user
	@sudo useradd -r -s /bin/false -d /opt/billaged -c "Billaged VM Usage Billing Service" billaged 2>/dev/null || true

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

dev: ## Run the service in development mode
	@go run ./cmd/billaged

env-example: ## Generate example environment file
	@echo "# Billaged Environment Variables" > .env.example
	@echo "UNKEY_BILLAGED_PORT=8081" >> .env.example
	@echo "UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s" >> .env.example
	@echo "UNKEY_BILLAGED_ASSET_MANAGER_ADDR=localhost:50052" >> .env.example
	@echo "UNKEY_BILLAGED_SPIFFE_ENABLED=false" >> .env.example
	@echo "✓ .env.example created"

fmt: ## Format Go code
	@goimports -w .

generate: ## Generate protobuf code
	@buf generate
	@buf lint

health: ## Check service health
	@curl -s http://localhost:8081/health | jq . || echo "Health check failed"

help: ## Show this help message
	@echo ""
	@echo "Billaged VM Usage Billing Service - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

install: build create-user ## Install billaged binary and systemd service
	@sudo systemctl stop billaged 2>/dev/null || true
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@sudo cp contrib/systemd/billaged.service /etc/systemd/system/billaged.service
	@sudo systemctl daemon-reload
	@sudo systemctl start billaged 2>/dev/null || true
	@echo "✓ billaged installed and started"

lint: lint-proto ## Run linting tools (includes protobuf linting)
	@which golangci-lint >/dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run --disable=godox

lint-proto: ## Run protobuf linter
	@buf lint

run: build ## Build and run the billaged service
	@./$(BUILD_DIR)/$(BINARY_NAME)

service-logs: ## Show all billaged service logs
	@sudo journalctl -u billaged --no-pager

service-logs-tail: ## Follow billaged service logs
	@sudo journalctl -u billaged -f

service-restart: ## Restart billaged service
	@sudo systemctl restart billaged
	@echo "✓ billaged restarted"

service-start: ## Start billaged service
	@sudo systemctl start billaged
	@echo "✓ billaged started"

service-status: ## Show billaged service status
	@sudo systemctl status billaged

service-stop: ## Stop billaged service
	@sudo systemctl stop billaged
	@echo "✓ billaged stopped"

setup: deps generate ## Complete development setup

test: ## Run all tests
	@go test ./... -v

test-coverage: ## Run tests with coverage report
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

uninstall: ## Uninstall billaged service and binary
	@sudo systemctl stop billaged 2>/dev/null || true
	@sudo systemctl disable billaged 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/billaged.service
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo systemctl daemon-reload
	@echo "✓ billaged uninstalled"

version: ## Show version information
	@echo "Billaged Version Information:"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"

vet: ## Run go vet
	@go vet ./...

