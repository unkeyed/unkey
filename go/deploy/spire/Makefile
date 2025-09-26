# SPIRE Installation Makefile
# Installs SPIRE server and agent as systemd services

# Variables
SPIRE_VERSION ?= 1.13.0
SPIRE_ARCH ?= linux-amd64-musl
SPIRE_URL = https://github.com/spiffe/spire/releases/download/v$(SPIRE_VERSION)/spire-$(SPIRE_VERSION)-$(SPIRE_ARCH).tar.gz
SPIRE_INSTALL_DIR = /opt/spire
SPIRE_DATA_DIR = /var/lib/spire
SPIRE_CONFIG_DIR = /etc/spire

# Environment selection (default to development)
SPIRE_ENVIRONMENT ?= development

# Validate environment
ifeq ($(filter $(SPIRE_ENVIRONMENT),development production),)
$(error Invalid SPIRE_ENVIRONMENT: $(SPIRE_ENVIRONMENT). Must be one of: development, production)
endif

# Trust domain mapping
ifeq ($(SPIRE_ENVIRONMENT),development)
TRUST_DOMAIN = development.unkey.cloud
else ifeq ($(SPIRE_ENVIRONMENT),production)
TRUST_DOMAIN = production.unkey.cliud
endif

# Targets (alphabetically ordered)

.PHONY: bootstrap-agent
bootstrap-agent: ## Bootstrap agent with server bundle
	@echo "Bootstrapping agent with server bundle..."
	@if ! sudo systemctl is-active --quiet spire-server; then \
		echo "Error: SPIRE server is not running. Start it first with 'make service-start-server'"; \
		exit 1; \
	fi
	@echo "Waiting for SPIRE server socket to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if [ -S /var/lib/spire/server/server.sock ]; then \
			echo "Server socket is ready"; \
			break; \
		fi; \
		echo "Waiting for server socket... ($$i/10)"; \
		sleep 2; \
	done
	@if [ ! -S /var/lib/spire/server/server.sock ]; then \
		echo "Error: Server socket not available after 20 seconds"; \
		exit 1; \
	fi
	@sudo bash -c '$(SPIRE_INSTALL_DIR)/bin/spire-server bundle show \
		-socketPath /var/lib/spire/server/server.sock \
		-format pem > /tmp/bootstrap.crt' || \
		(echo "Error: Failed to get bundle from server. Check server logs with 'make service-logs-server'" && exit 1)
	@sudo mv /tmp/bootstrap.crt $(SPIRE_CONFIG_DIR)/agent/bootstrap.crt
	@sudo chmod 600 $(SPIRE_CONFIG_DIR)/agent/bootstrap.crt
	@echo "Agent bootstrap bundle created."

.PHONY: clean
clean: ## Clean up temporary files
	@rm -rf /tmp/spire-install

.PHONY: clean-all
clean-all: ## Completely remove SPIRE (data, configs)
	@echo "Stopping and removing SPIRE services..."
	@sudo systemctl stop spire-server spire-agent 2>/dev/null || true
	@sudo systemctl disable spire-server spire-agent 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/spire-server.service
	@sudo rm -f /etc/systemd/system/spire-agent.service
	@sudo rm -rf /etc/systemd/system/spire-server.service.d
	@sudo rm -rf /etc/systemd/system/spire-agent.service.d
	@sudo systemctl daemon-reload
	@echo "Removing SPIRE binaries..."
	@sudo rm -rf $(SPIRE_INSTALL_DIR)
	@echo "Removing SPIRE data and configs..."
	@sudo rm -rf $(SPIRE_DATA_DIR)
	@sudo rm -rf $(SPIRE_CONFIG_DIR)
	@echo "Removing SPIRE users..."
	@sudo userdel -r spire-server 2>/dev/null || true
	@sudo userdel -r spire-agent 2>/dev/null || true
	@echo "SPIRE completely removed."

.PHONY: create-join-token
create-join-token: ## Create join token for agent
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server token generate \
		-socketPath /var/lib/spire/server/server.sock \
		-spiffeID spiffe://$(TRUST_DOMAIN)/agent/node1

.PHONY: deregister-services
deregister-services: ## Deregister all Unkey services from SPIRE
	@TRUST_DOMAIN=$(TRUST_DOMAIN) ./scripts/deregister-services.sh

.PHONY: download-spire
download-spire: ## Download SPIRE binaries
	@if [ ! -f /tmp/spire-install/spire-$(SPIRE_VERSION)/bin/spire-agent ]; then \
		rm -rf /tmp/spire-install/spire-$(SPIRE_VERSION); \
		mkdir -p /tmp/spire-install; \
		curl -sL $(SPIRE_URL) | tar xz -C /tmp/spire-install; \
	fi

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Available environments: development, production"

.PHONY: install
install: install-server install-agent ## Install both SPIRE server and agent
	@echo "Next steps: 'make service-start-server', then 'make register-agent', then 'make register-services'"

.PHONY: install-agent
install-agent: download-spire setup-directories ## Install SPIRE agent
	@sudo systemctl stop spire-agent 2>/dev/null || true
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-agent
	@sudo cp contrib/systemd/spire-agent.service /etc/systemd/system/spire-agent.service
	@sudo cp /tmp/spire-install/spire-$(SPIRE_VERSION)/bin/spire-agent $(SPIRE_INSTALL_DIR)/bin/
	@sudo chmod +x $(SPIRE_INSTALL_DIR)/bin/spire-agent
	@sudo cp contrib/bin/spire-agent-wrapper.sh $(SPIRE_INSTALL_DIR)/bin/
	@sudo chmod +x $(SPIRE_INSTALL_DIR)/bin/spire-agent-wrapper.sh
	@sudo cp environments/$(SPIRE_ENVIRONMENT)/agent.conf $(SPIRE_CONFIG_DIR)/agent/agent.conf
	@sudo chmod 700 $(SPIRE_DATA_DIR)/agent/keys
	@sudo mkdir -p /etc/systemd/system/spire-agent.service.d
	@sudo cp contrib/systemd/spire-agent.service.d/auto-join.conf /etc/systemd/system/spire-agent.service.d/
	@sudo systemctl daemon-reload
	@sudo systemctl enable spire-agent >/dev/null 2>&1
	@echo "✓ SPIRE agent installed ($(SPIRE_ENVIRONMENT): $(TRUST_DOMAIN))"
	@echo "Next: Start server with 'make service-start-server', then 'make register-agent'"

.PHONY: install-server
install-server: download-spire create-users setup-directories ## Install SPIRE server
	@sudo systemctl stop spire-server 2>/dev/null || true
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-server
	@sudo cp /tmp/spire-install/spire-$(SPIRE_VERSION)/bin/spire-server $(SPIRE_INSTALL_DIR)/bin/
	@sudo chmod +x $(SPIRE_INSTALL_DIR)/bin/spire-server
	@sudo cp environments/$(SPIRE_ENVIRONMENT)/server.conf $(SPIRE_CONFIG_DIR)/server/server.conf
	@sudo cp contrib/systemd/spire-server.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable spire-server >/dev/null 2>&1
	@echo "✓ SPIRE server installed ($(SPIRE_ENVIRONMENT): $(TRUST_DOMAIN))"

.PHONY: list-entries
list-entries: ## List all registered entries
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server entry show \
		-socketPath /var/lib/spire/server/server.sock

.PHONY: register-agent
register-agent: ## Register agent with join token (one-time setup)
	@./scripts/register-agent.sh

.PHONY: register-services
register-services: ## Register all Unkey services with SPIRE
	@TRUST_DOMAIN=$(TRUST_DOMAIN) ./scripts/register-services.sh

.PHONY: service-logs
service-logs: ## Follow logs for both services
	@sudo journalctl -u spire-server -u spire-agent -f

.PHONY: service-logs-agent
service-logs-agent: ## Follow SPIRE agent logs
	@sudo journalctl -u spire-agent -f

.PHONY: service-logs-server
service-logs-server: ## Follow SPIRE server logs
	@sudo journalctl -u spire-server -f

.PHONY: service-restart
service-restart: service-restart-server service-restart-agent ## Restart both

.PHONY: service-restart-agent
service-restart-agent: ## Restart SPIRE agent
	@sudo systemctl restart spire-agent
	@echo "✓ SPIRE agent restarted"

.PHONY: service-restart-server
service-restart-server: ## Restart SPIRE server
	@sudo systemctl restart spire-server
	@echo "✓ SPIRE server restarted"

.PHONY: service-start
service-start: service-start-server service-start-agent ## Start both server and agent

.PHONY: service-start-agent
service-start-agent: ## Start SPIRE agent
	@sudo systemctl start spire-agent
	@echo "✓ SPIRE agent started"

.PHONY: service-start-server
service-start-server: ## Start SPIRE server
	@sudo systemctl start spire-server
	@echo "✓ SPIRE server started"

.PHONY: service-status
service-status: ## Check status of both services
	@echo "=== SPIRE Server ==="
	@sudo systemctl status spire-server --no-pager || true
	@echo ""
	@echo "=== SPIRE Agent ==="
	@sudo systemctl status spire-agent --no-pager || true

.PHONY: service-status-agent
service-status-agent: ## Check SPIRE agent status
	@sudo systemctl status spire-agent

.PHONY: service-status-server
service-status-server: ## Check SPIRE server status
	@sudo systemctl status spire-server

.PHONY: service-stop
service-stop: service-stop-agent service-stop-server ## Stop both agent and server

.PHONY: service-stop-agent
service-stop-agent: ## Stop SPIRE agent
	@sudo systemctl stop spire-agent
	@echo "✓ SPIRE agent stopped"

.PHONY: service-stop-server
service-stop-server: ## Stop SPIRE server
	@sudo systemctl stop spire-server
	@echo "✓ SPIRE server stopped"

.PHONY: setup-agent
setup-agent: ## Setup agent with join token
	@./scripts/register-agent.sh

.PHONY: setup-directories
setup-directories: ## Create SPIRE directories
	@sudo mkdir -p $(SPIRE_INSTALL_DIR)/bin
	@sudo mkdir -p $(SPIRE_INSTALL_DIR)/scripts
	@sudo mkdir -p $(SPIRE_CONFIG_DIR)/server
	@sudo mkdir -p $(SPIRE_CONFIG_DIR)/agent
	@sudo mkdir -p $(SPIRE_DATA_DIR)/server
	@sudo mkdir -p $(SPIRE_DATA_DIR)/agent
	@sudo mkdir -p $(SPIRE_DATA_DIR)/agent/keys

.PHONY: uninstall
uninstall: uninstall-server uninstall-agent ## Uninstall both SPIRE server and agent

.PHONY: uninstall-agent
uninstall-agent: ## Uninstall SPIRE agent
	@sudo systemctl stop spire-agent 2>/dev/null || true
	@sudo systemctl disable spire-agent 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/spire-agent.service
	@sudo rm -rf /etc/systemd/system/spire-agent.service.d
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-agent
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-agent-wrapper.sh
	@sudo systemctl daemon-reload
	@echo "✓ SPIRE agent uninstalled (data preserved)"

.PHONY: uninstall-server
uninstall-server: ## Uninstall SPIRE server
	@sudo systemctl stop spire-server 2>/dev/null || true
	@sudo systemctl disable spire-server 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/spire-server.service
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-server
	@sudo systemctl daemon-reload
	@echo "✓ SPIRE server uninstalled (data preserved)"
