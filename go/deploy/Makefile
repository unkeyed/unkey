# Deploy Services Makefile
# Calls metald and billaged Makefiles

.DEFAULT_GOAL := help

.PHONY: help build test clean lint generate run dev deps install uninstall service-start service-stop service-restart service-status service-logs health setup-metald setup-billaged setup o11y o11y-stop o11y-restart o11y-logs clean-all

# Variables
METALD_DIR = metald
BILLAGED_DIR = billaged
SPIRE_DIR = spire

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

# Observability stack
o11y: ## Start observability stack (Grafana LGTM)
	@echo "Starting Grafana + OpenTelemetry LGTM stack..."
	@docker run -d \
		--name unkey-o11y-lgtm \
		-p 3000:3000 \
		-p 4317:4317 \
		-p 4318:4318 \
		-p 9090:9090 \
		-p 3100:3100 \
		-e GF_SERVER_HTTP_ADDR=0.0.0.0 \
		grafana/otel-lgtm:latest
	@echo ""
	@echo "==================================="
	@echo "✓ Observability stack started!"
	@echo "==================================="
	@echo "Grafana UI: http://0.0.0.0:3000 (admin/admin)"
	@echo "OTLP gRPC: 0.0.0.0:4317"
	@echo "OTLP HTTP: 0.0.0.0:4318"
	@echo "Prometheus: http://0.0.0.0:9090"
	@echo "Loki: http://0.0.0.0:3100"
	@echo ""

o11y-stop: ## Stop observability stack
	@echo "Stopping observability stack..."
	@docker stop unkey-o11y-lgtm 2>/dev/null || true
	@docker rm unkey-o11y-lgtm 2>/dev/null || true
	@echo "✓ Observability stack stopped"

o11y-logs: ## Show observability stack logs
	@docker logs -f unkey-o11y-lgtm

o11y-restart: ## Restart observability stack
	@$(MAKE) o11y-stop
	@sleep 2
	@$(MAKE) o11y

# SPIRE targets
spire-install: ## Install SPIRE server and agent
	$(MAKE) -C $(SPIRE_DIR) install

spire-uninstall: ## Uninstall SPIRE server and agent
	$(MAKE) -C $(SPIRE_DIR) uninstall

spire-start: ## Start SPIRE services and register Unkey services
	$(MAKE) -C $(SPIRE_DIR) service-start-server
	@sleep 3
	$(MAKE) -C $(SPIRE_DIR) bootstrap-agent
	$(MAKE) -C $(SPIRE_DIR) register-agent
	$(MAKE) -C $(SPIRE_DIR) service-start-agent
	@sleep 2
	$(MAKE) -C $(SPIRE_DIR) register-services

spire-stop: ## Stop SPIRE services
	$(MAKE) -C $(SPIRE_DIR) service-stop

spire-status: ## Check SPIRE service status
	$(MAKE) -C $(SPIRE_DIR) service-status

spire-logs: ## Follow SPIRE service logs
	$(MAKE) -C $(SPIRE_DIR) service-logs

spire-%: ## Run any SPIRE target (e.g., make spire-list-entries)
	$(MAKE) -C $(SPIRE_DIR) $*

# Complete system cleanup
clean-all: ## Complete uninstall and cleanup of all services and data
	@echo "==================================="
	@echo "Complete System Cleanup"
	@echo "==================================="
	@echo "This will:"
	@echo "- Stop all running services"
	@echo "- Uninstall all service binaries"
	@echo "- Remove all service users"
	@echo "- Delete all service data"
	@echo "- Remove SPIRE installation"
	@echo "- Stop observability stack"
	@echo ""
	@read -p "Are you sure you want to completely clean the system? [y/N] " -n 1 -r; \
	echo ""; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Starting complete cleanup..."; \
		$(MAKE) clean-all-force; \
	else \
		echo "Cleanup cancelled."; \
	fi

clean-all-force: ## Force complete cleanup without confirmation
	@echo "Stopping all services..."
	@for service in metald builderd billaged assetmanagerd; do \
		echo "Stopping $$service..."; \
		sudo systemctl stop $$service 2>/dev/null || true; \
	done
	@echo "Stopping SPIRE..."
	@sudo systemctl stop spire-agent 2>/dev/null || true
	@sudo systemctl stop spire-server 2>/dev/null || true
	@echo "Stopping observability stack..."
	@$(MAKE) o11y-stop 2>/dev/null || true
	@echo ""
	@echo "Uninstalling services..."
	@for service in metald builderd billaged assetmanagerd; do \
		echo "Uninstalling $$service..."; \
		$(MAKE) -C $$service uninstall 2>/dev/null || true; \
	done
	@echo "Uninstalling SPIRE..."
	@$(MAKE) -C $(SPIRE_DIR) uninstall 2>/dev/null || true
	@echo ""
	@echo "Removing service users..."
	@for user in metald billaged builderd assetmanagerd spire-server spire-agent; do \
		echo "Removing user $$user..."; \
		sudo userdel -r $$user 2>/dev/null || true; \
	done
	@echo ""
	@echo "Removing service directories..."
	@sudo rm -rf /opt/metald /opt/billaged /opt/builderd /opt/assetmanagerd /opt/vm-assets
	@sudo rm -rf /opt/spire /var/lib/spire /etc/spire /run/spire
	@echo ""
	@echo "Cleaning up systemd..."
	@sudo systemctl daemon-reload
	@echo ""
	@echo "Cleaning build artifacts..."
	@for service in metald billaged builderd assetmanagerd; do \
		$(MAKE) -C $$service clean 2>/dev/null || true; \
	done
	@$(MAKE) -C $(SPIRE_DIR) clean 2>/dev/null || true
	@echo ""
	@echo "==================================="
	@echo "✓ System cleanup complete!"
	@echo "==================================="