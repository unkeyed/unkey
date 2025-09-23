# Makefile for metald-init - statically compiled init wrapper

# Build variables
VERSION ?= 0.1.0
BUILD_TIME := $(shell date -u +"%Y-%m-%d %H:%M:%S UTC")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X 'main.buildTime=$(BUILD_TIME)' -extldflags '-static'"

# Output binary name
BINARY := metald-init

# Default target
.DEFAULT_GOAL := build

# Targets (alphabetically ordered)

.PHONY: build
build: ## Build static binary for linux/amd64
	@echo "Building $(BINARY) v$(VERSION) (static)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY) .
	@echo "Built $(BINARY) successfully"

.PHONY: build-all
build-all: build-amd64 build-arm64 ## Build for multiple architectures

.PHONY: build-amd64
build-amd64: ## Build for linux/amd64
	@echo "Building $(BINARY) for linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .

.PHONY: build-arm64
build-arm64: ## Build for linux/arm64
	@echo "Building $(BINARY) for linux/arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .

.PHONY: clean
clean: ## Remove build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY) $(BINARY)-linux-*

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@echo "  build        - Build static binary for linux/amd64 (default)"
	@echo "  build-all    - Build for multiple architectures"
	@echo "  build-amd64  - Build for linux/amd64"
	@echo "  build-arm64  - Build for linux/arm64"
	@echo "  clean        - Remove build artifacts"
	@echo "  fmt          - Format code"
	@echo "  info         - Show binary information"
	@echo "  install      - Install to /usr/bin"
	@echo "  lint         - Run linter"
	@echo "  test         - Run basic tests"
	@echo "  test-echo    - Test with echo command"
	@echo "  test-env     - Test with environment variables"
	@echo "  help         - Show this help"

.PHONY: info
info: build ## Show binary information
	@echo "Binary information:"
	@file $(BINARY)
	@ls -lh $(BINARY)
	@ldd $(BINARY) 2>/dev/null || echo "Binary is statically linked (no dynamic dependencies)"

.PHONY: install
install: build ## Install to /usr/local/bin
	@echo "Installing $(BINARY) to /usr/bin..."
	sudo cp $(BINARY) /usr/bin/
	sudo chmod +x /usr/bin/$(BINARY)

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

.PHONY: test-echo
test-echo: build ## Test with echo command
	./$(BINARY) -- echo "Hello from metald-init"
