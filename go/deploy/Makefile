# Deploy Services Makefile
# Calls metald and billaged Makefiles

.DEFAULT_GOAL := help

.PHONY: help build test clean lint generate run dev deps install uninstall service-start service-stop service-restart service-status service-logs health setup-metald setup-billaged setup

# Variables
METALD_DIR = metald
BILLAGED_DIR = billaged

# Help target
help: ## Show this help message
	@echo ""
	@echo "Deploy Services - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Individual service help:"
	@echo "  make metald-help     Show metald targets"
	@echo "  make billaged-help   Show billaged targets"
	@echo ""

# Build targets
build: metald-build billaged-build ## Build both metald and billaged services

# Development targets
deps: metald-deps billaged-deps ## Download dependencies for both services

generate: metald-generate billaged-generate ## Generate protobuf code for both services

# Code quality targets
test: metald-test billaged-test ## Run tests for both services

lint: metald-lint billaged-lint ## Run linting for both services

fmt: metald-fmt billaged-fmt ## Format code for both services

vet: metald-vet billaged-vet ## Run go vet for both services

# Installation targets
install: metald-install billaged-install ## Install both services

uninstall: metald-uninstall billaged-uninstall ## Uninstall both services

# Service management
service-start: metald-service-start billaged-service-start ## Start both services

service-stop: metald-service-stop billaged-service-stop ## Stop both services

service-restart: metald-service-restart billaged-service-restart ## Restart both services

service-status: metald-service-status billaged-service-status ## Show status of both services

service-logs: ## Follow logs for both services in parallel
	@echo "Following logs for both services (Ctrl+C to stop)..."
	$(MAKE) -C $(METALD_DIR) service-logs &
	$(MAKE) -C $(BILLAGED_DIR) service-logs &
	wait

# Health checks
health: metald-health billaged-health ## Check health of both services

# Clean targets
clean: metald-clean billaged-clean ## Clean build artifacts for both services

# Setup targets
setup-metald: ## Setup metald development environment
	$(MAKE) -C $(METALD_DIR) setup

setup-billaged: ## Setup billaged development environment
	$(MAKE) -C $(BILLAGED_DIR) setup

setup: setup-metald setup-billaged ## Complete development setup for both services

# Individual service help
metald-help: ## Show metald Makefile help
	$(MAKE) -C $(METALD_DIR) help

billaged-help: ## Show billaged Makefile help
	$(MAKE) -C $(BILLAGED_DIR) help

# Individual service targets
metald-%: ## Run any metald target (e.g., make metald-build)
	$(MAKE) -C $(METALD_DIR) $*

billaged-%: ## Run any billaged target (e.g., make billaged-build)
	$(MAKE) -C $(BILLAGED_DIR) $*