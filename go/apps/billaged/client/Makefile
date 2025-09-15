# Makefile for billaged CLI client

# Variables
BINARY_NAME := billaged-cli
BUILD_DIR := build
VERSION ?= 0.5.1

# Default target
.DEFAULT_GOAL := help

# Targets (alphabetically ordered)

.PHONY: build
build: ## Build the billaged CLI client
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ../cmd/billaged-cli/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@echo "  build   - Build the billaged CLI client"
	@echo "  clean   - Clean build artifacts"
	@echo "  install - Install the CLI client to /usr/local/bin"
	@echo "  help    - Show this help message"

.PHONY: install
install: build ## Install the CLI client to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo mv $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete"