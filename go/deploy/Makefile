# Deploy Services Makefile
# Calls metald and billaged Makefiles

.DEFAULT_GOAL := help

# Variables
SERVICES = assetmanagerd billaged builderd metald
METALD_DIR = metald
BILLAGED_DIR = billaged
ASSETMANAGERD_DIR = assetmanagerd
BUILDERD_DIR = builderd
SPIRE_DIR = spire

# Individual service targets (pattern rules - must come before other targets)
assetmanagerd-%: ## Run any assetmanagerd target (e.g., make assetmanagerd-build)
	$(MAKE) -C $(ASSETMANAGERD_DIR) $*

billaged-%: ## Run any billaged target (e.g., make billaged-build)
	$(MAKE) -C $(BILLAGED_DIR) $*

builderd-%: ## Run any builderd target (e.g., make builderd-build)
	$(MAKE) -C $(BUILDERD_DIR) $*

metald-%: ## Run any metald target (e.g., make metald-build)
	$(MAKE) -C $(METALD_DIR) $*

spire-%: ## Run any SPIRE target (e.g., make spire-list-entries)
	$(MAKE) -C $(SPIRE_DIR) $*

# Targets (alphabetically ordered)
.PHONY: billaged-help build check clean clean-all clean-all-force deps fmt generate health help install install-tools lint metald-help o11y o11y-logs o11y-restart o11y-stop service-logs service-restart service-start service-status service-stop setup setup-billaged setup-metald spire-install spire-logs spire-start spire-status spire-stop spire-uninstall test uninstall vet

billaged-help: ## Show billaged Makefile help
	$(MAKE) -C $(BILLAGED_DIR) help

build: assetmanagerd-build builderd-build metald-build billaged-build ## Build both metald and billaged services

check: fmt vet lint test ## Run all checks (fmt, vet, lint with proto, test)

clean: metald-clean billaged-clean ## Clean build artifacts for both services

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
	@for service in $(SERVICES); do \
		sudo systemctl stop $$service 2>/dev/null || true; \
	done
	@sudo systemctl stop spire-agent 2>/dev/null || true
	@sudo systemctl stop spire-server 2>/dev/null || true
	@$(MAKE) o11y-stop 2>/dev/null || true
	@for service in $(SERVICES); do \
		$(MAKE) -C $$service uninstall 2>/dev/null || true; \
	done
	@$(MAKE) -C $(SPIRE_DIR) uninstall 2>/dev/null || true
	@for user in $(SERVICES); do \
		if [ "$$user" != "metald" ]; then \
			sudo userdel -r $$user 2>/dev/null || true; \
		fi; \
	done
	@for service in $(SERVICES); do \
		sudo rm -rf /opt/$$service; \
	done
	@sudo rm -rf /opt/vm-assets
	@sudo rm -rf /opt/spire /var/lib/spire /etc/spire /run/spire
	@sudo systemctl daemon-reload
	@for service in $(SERVICES); do \
		$(MAKE) -C $$service clean 2>/dev/null || true; \
	done
	@$(MAKE) -C $(SPIRE_DIR) clean 2>/dev/null || true
	@echo "✓ System cleanup complete"

deps: assetmanagerd-deps builderd-deps metald-deps billaged-deps ## Download dependencies for both services

fmt: assetmanagerd-fmt builderd-fmt metald-fmt billaged-fmt ## Format code for both services

generate: assetmanagerd-generate builderd-generate metald-generate billaged-generate ## Generate protobuf code for both services

health: metald-health billaged-health ## Check health of both services

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

install: assetmanagerd-install builderd-install metald-install billaged-install ## Install both services

install-tools: ## Install development tools
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@command buf 2>/dev/null || (sudo /usr/bin/curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$$(uname -s)-$$(uname -m)" -o /usr/local/bin/buf && sudo chmod +x /usr/local/bin/buf)
	@echo "✓ Development tools installed"

lint: assetmanagerd-lint builderd-lint metald-lint billaged-lint ## Run linting for both services

metald-help: ## Show metald Makefile help
	$(MAKE) -C $(METALD_DIR) help

o11y: ## Start observability stack (Grafana LGTM)
	@docker run -d \
		--name unkey-o11y-lgtm \
		-p 3000:3000 \
		-p 4317:4317 \
		-p 4318:4318 \
		-p 9090:9090 \
		-p 3100:3100 \
		-e GF_SERVER_HTTP_ADDR=0.0.0.0 \
		grafana/otel-lgtm:latest >/dev/null
	@echo "✓ Observability stack started"
	@echo "Grafana UI: http://0.0.0.0:3000 (admin/admin)"
	@echo "OTLP gRPC: 0.0.0.0:4317 | OTLP HTTP: 0.0.0.0:4318"
	@echo "Prometheus: http://0.0.0.0:9090 | Loki: http://0.0.0.0:3100"

o11y-logs: ## Show observability stack logs
	@docker logs -f unkey-o11y-lgtm

o11y-restart: ## Restart observability stack
	@$(MAKE) o11y-stop
	@sleep 2
	@$(MAKE) o11y

o11y-stop: ## Stop observability stack
	@docker stop unkey-o11y-lgtm 2>/dev/null || true
	@docker rm unkey-o11y-lgtm 2>/dev/null || true
	@echo "✓ Observability stack stopped"

service-logs: ## Follow logs for both services in parallel
	@$(MAKE) -C $(METALD_DIR) service-logs &
	@$(MAKE) -C $(BILLAGED_DIR) service-logs &
	@wait

service-restart: metald-service-restart billaged-service-restart ## Restart both services

service-start: metald-service-start billaged-service-start ## Start both services

service-status: assetmanagerd-service-status builderd-service-status metald-service-status billaged-service-status ## Show status of both services

service-stop: metald-service-stop billaged-service-stop ## Stop both services

setup: setup-metald setup-billaged ## Complete development setup for both services

setup-billaged: ## Setup billaged development environment
	$(MAKE) -C $(BILLAGED_DIR) setup

setup-metald: ## Setup metald development environment
	$(MAKE) -C $(METALD_DIR) setup

spire-install: ## Install SPIRE server and agent
	$(MAKE) -C $(SPIRE_DIR) install

spire-logs: ## Follow SPIRE service logs
	$(MAKE) -C $(SPIRE_DIR) service-logs

spire-start: ## Start SPIRE services and register Unkey services
	$(MAKE) -C $(SPIRE_DIR) service-start-server
	@sleep 3
	$(MAKE) -C $(SPIRE_DIR) bootstrap-agent
	$(MAKE) -C $(SPIRE_DIR) register-agent
	$(MAKE) -C $(SPIRE_DIR) service-start-agent
	@sleep 2
	$(MAKE) -C $(SPIRE_DIR) register-services

spire-status: ## Check SPIRE service status
	$(MAKE) -C $(SPIRE_DIR) service-status

spire-stop: ## Stop SPIRE services
	$(MAKE) -C $(SPIRE_DIR) service-stop

spire-uninstall: ## Uninstall SPIRE server and agent
	$(MAKE) -C $(SPIRE_DIR) uninstall

test: assetmanagerd-test builderd-test metald-test billaged-test ## Run tests for both services

uninstall: assetmanagerd-uninstall builderd-uninstall metald-uninstall billaged-uninstall ## Uninstall both services

vet: assetmanagerd-vet builderd-vet metald-vet billaged-vet ## Run go vet for both services
