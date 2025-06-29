# Makefile for metald-init - statically compiled init wrapper

# Build variables
VERSION ?= 0.1.0
BUILD_TIME := $(shell date -u +"%Y-%m-%d %H:%M:%S UTC")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X 'main.buildTime=$(BUILD_TIME)' -extldflags '-static'"

# Output binary name
BINARY := metald-init

# Default target
.PHONY: all
all: build

# Build the binary (statically linked)
.PHONY: build
build:
	@echo "Building $(BINARY) v$(VERSION) (static)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY) .
	@echo "Built $(BINARY) successfully"

# Build for multiple architectures
.PHONY: build-all
build-all: build-amd64 build-arm64

.PHONY: build-amd64
build-amd64:
	@echo "Building $(BINARY) for linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .

.PHONY: build-arm64
build-arm64:
	@echo "Building $(BINARY) for linux/arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .

# Test the binary locally
.PHONY: test
test: build
	@echo "Testing $(BINARY)..."
	./$(BINARY) --version
	./$(BINARY) --help

# Test with a simple command
.PHONY: test-echo
test-echo: build
	./$(BINARY) -- echo "Hello from metald-init"

# Test with environment variables
.PHONY: test-env
test-env: build
	@echo "Testing with mock kernel cmdline..."
	@# This would need to be run in a VM or with a mock /proc/cmdline

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f $(BINARY) $(BINARY)-linux-*

# Show binary info
.PHONY: info
info: build
	@echo "Binary information:"
	@file $(BINARY)
	@ls -lh $(BINARY)
	@ldd $(BINARY) 2>/dev/null || echo "Binary is statically linked (no dynamic dependencies)"

# Install to system (for testing in VMs)
.PHONY: install
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	sudo cp $(BINARY) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Build the binary (default)"
	@echo "  build        - Build static binary for linux/amd64"
	@echo "  build-all    - Build for multiple architectures"
	@echo "  test         - Run basic tests"
	@echo "  test-echo    - Test with echo command"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  clean        - Remove build artifacts"
	@echo "  info         - Show binary information"
	@echo "  install      - Install to /usr/local/bin"
	@echo "  help         - Show this help"