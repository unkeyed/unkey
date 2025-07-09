# Ctrl Control Plane Service - Docker Makefile

.DEFAULT_GOAL := help

# Variables
SERVICE_NAME := ctrl
IMAGE_NAME := unkey/$(SERVICE_NAME)
TRUST_DOMAIN := development.unkey.app
SPIRE_SOCKET := /var/lib/spire/server/server.sock
SPIRE_DIR := /opt/spire

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
BLUE := \033[0;34m
CYAN := \033[36m
NC := \033[0m
RESET := \033[0m

.PHONY: help build run stop logs spire-register clean compose-up compose-down

help: ## Display this help message
	@echo ""
	@echo "$(CYAN)Ctrl Control Plane Service - Docker Management$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(CYAN)<target>$(RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(IMAGE_NAME):latest --build-arg VERSION=latest ../../
	@echo "$(GREEN)✓ Image built: $(IMAGE_NAME):latest$(NC)"

run: ## Run Docker container
	@echo "$(BLUE)Starting ctrl container...$(NC)"
	@docker run -d --name unkey-$(SERVICE_NAME) \
		-p 8084:8084 \
		-v /var/lib/spire/agent/agent.sock:/var/lib/spire/agent/agent.sock \
		-e UNKEY_DATABASE_PRIMARY="$${UNKEY_DATABASE_PRIMARY}" \
		-e UNKEY_DATABASE_HYDRA="$${UNKEY_DATABASE_HYDRA}" \
		-e UNKEY_METALD_ADDRESS="$${UNKEY_METALD_ADDRESS:-https://host.docker.internal:8080}" \
		-e UNKEY_HTTP_PORT="8084" \
		-e UNKEY_PLATFORM="docker" \
		-e UNKEY_REGION="docker" \
		-e UNKEY_OTEL="true" \
		-e UNKEY_SPIFFE_SOCKET_PATH="/var/lib/spire/agent/agent.sock" \
		$(IMAGE_NAME):latest run ctrl
	@echo "$(GREEN)✓ Container started: unkey-$(SERVICE_NAME)$(NC)"
	@echo "$(YELLOW)Next: Run 'make spire-register' to register with SPIRE$(NC)"

stop: ## Stop and remove Docker container
	@echo "$(BLUE)Stopping ctrl container...$(NC)"
	@docker stop unkey-$(SERVICE_NAME) 2>/dev/null || true
	@docker rm unkey-$(SERVICE_NAME) 2>/dev/null || true
	@echo "$(GREEN)✓ Container stopped$(NC)"

logs: ## Follow container logs
	@docker logs -f unkey-$(SERVICE_NAME)

spire-register: ## Register Docker container with SPIRE
	@echo "$(BLUE)Registering ctrl container with SPIRE...$(NC)"
	@echo "Trust Domain: $(YELLOW)$(TRUST_DOMAIN)$(NC)"
	@if [ ! -S "$(SPIRE_SOCKET)" ]; then \
		echo "$(RED)Error: SPIRE server socket not available at $(SPIRE_SOCKET)$(NC)"; \
		exit 1; \
	fi
	@CONTAINER_ID=$$(docker ps -q -f name=unkey-$(SERVICE_NAME)); \
	if [ -z "$$CONTAINER_ID" ]; then \
		echo "$(RED)Error: ctrl container not running$(NC)"; \
		echo "Start it with: make run"; \
		exit 1; \
	fi; \
	echo "$(BLUE)Container ID: $$CONTAINER_ID$(NC)"; \
	PARENT_ID="spiffe://$(TRUST_DOMAIN)/agent/node1"; \
	SPIFFE_ID="spiffe://$(TRUST_DOMAIN)/service/$(SERVICE_NAME)"; \
	if sudo $(SPIRE_DIR)/bin/spire-server entry show -socketPath "$(SPIRE_SOCKET)" -spiffeID "$$SPIFFE_ID" 2>/dev/null | grep -q "SPIFFE ID"; then \
		echo "$(YELLOW)✓ $(SERVICE_NAME) already registered$(NC)"; \
	else \
		IMAGE_ID=$$(docker inspect --format='{{.Image}}' $$CONTAINER_ID); \
		sudo $(SPIRE_DIR)/bin/spire-server entry create \
			-socketPath "$(SPIRE_SOCKET)" \
			-parentID "$$PARENT_ID" \
			-spiffeID "$$SPIFFE_ID" \
			-selector "docker:label:com.docker.compose.service:$(SERVICE_NAME)" \
			-selector "docker:image_id:$$IMAGE_ID" \
			-x509SVIDTTL 3600 \
			&& echo "$(GREEN)✓ $(SERVICE_NAME) registered with SPIRE$(NC)" \
			|| (echo "$(RED)✗ Failed to register $(SERVICE_NAME)$(NC)" && exit 1); \
	fi

clean: ## Remove Docker image
	@echo "$(BLUE)Cleaning up...$(NC)"
	@docker rmi $(IMAGE_NAME):latest 2>/dev/null || true
	@echo "$(GREEN)✓ Cleanup complete$(NC)"

# Combined targets
up: build run spire-register ## Build, run, and register with SPIRE
	@echo "$(GREEN)✓ Ctrl service is up and running!$(NC)"

down: stop ## Stop container
	@echo "$(GREEN)✓ Ctrl service stopped$(NC)"

compose-up: ## Build and start with docker-compose
	@echo "$(BLUE)Building and starting with docker-compose...$(NC)"
	@docker compose up -d --build
	@echo "$(GREEN)✓ Container built and started with docker-compose$(NC)"
	@echo "$(YELLOW)Next: Run 'make spire-register' to register with SPIRE$(NC)"

compose-down: ## Stop docker-compose
	@echo "$(BLUE)Stopping docker-compose...$(NC)"
	@docker compose down
	@echo "$(GREEN)✓ Docker-compose stopped$(NC)"
