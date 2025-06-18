# AIDEV-NOTE: Makefile for building metald example client
# Follows the same patterns as the main service Makefiles

.PHONY: all build clean run-dev run-spiffe help

# Binary name
BINARY := metald-client

# Build variables
GO := go
GOFLAGS := -v
LDFLAGS := -w -s

# Default target
all: build

# Build the client
build:
	@echo "Building $(BINARY)..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo "Build complete: ./$(BINARY)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY)
	$(GO) clean -cache
	@echo "Clean complete"

# Run in development mode (no TLS)
run-dev: build
	./$(BINARY) \
		--addr "http://localhost:8080" \
		--customer "dev-customer" \
		--tls-mode disabled

# Run with SPIFFE/SPIRE
run-spiffe: build
	./$(BINARY) \
		--addr "https://localhost:8080" \
		--customer "example-customer" \
		--tls-mode spiffe

# Run listing VMs
list-vms: build
	./$(BINARY) \
		--addr "http://localhost:8080" \
		--customer "dev-customer" \
		--tls-mode disabled \
		--action list

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Verify dependencies
verify:
	$(GO) mod verify

# Help target
help:
	@echo "Available targets:"
	@echo "  build       - Build the example client"
	@echo "  clean       - Remove build artifacts"
	@echo "  run-dev     - Run client in development mode (no TLS)"
	@echo "  run-spiffe  - Run client with SPIFFE/SPIRE"
	@echo "  list-vms    - List all VMs"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  verify      - Verify dependencies"
	@echo "  help        - Show this help message"