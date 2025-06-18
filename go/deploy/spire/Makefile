# SPIRE Installation Makefile
# Installs SPIRE server and agent as systemd services

.PHONY: help install install-server install-agent uninstall uninstall-server uninstall-agent
.PHONY: service-start service-stop service-restart service-status service-logs
.PHONY: download-spire setup-directories create-users

# Variables
SPIRE_VERSION ?= 1.12.2
SPIRE_ARCH ?= linux-amd64-musl
SPIRE_URL = https://github.com/spiffe/spire/releases/download/v$(SPIRE_VERSION)/spire-$(SPIRE_VERSION)-$(SPIRE_ARCH).tar.gz
SPIRE_INSTALL_DIR = /opt/spire
SPIRE_DATA_DIR = /var/lib/spire
SPIRE_CONFIG_DIR = /etc/spire

# Environment selection (default to development)
SPIRE_ENVIRONMENT ?= dev

# Validate environment
ifeq ($(filter $(SPIRE_ENVIRONMENT),dev canary prod),)
$(error Invalid SPIRE_ENVIRONMENT: $(SPIRE_ENVIRONMENT). Must be one of: dev, canary, prod)
endif

# Trust domain mapping
ifeq ($(SPIRE_ENVIRONMENT),dev)
TRUST_DOMAIN = development.unkey.app
else ifeq ($(SPIRE_ENVIRONMENT),canary)
TRUST_DOMAIN = canary.unkey.app
else ifeq ($(SPIRE_ENVIRONMENT),prod)
TRUST_DOMAIN = prod.unkey.app
endif

help: ## Show this help message
	@echo "SPIRE Installation Targets:"
	@echo ""
	@echo "Current environment: \033[32m$(SPIRE_ENVIRONMENT)\033[0m (trust domain: \033[32m$(TRUST_DOMAIN)\033[0m)"
	@echo "To change environment: SPIRE_ENVIRONMENT=prod make <target>"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Available environments: dev, canary, prod"

download-spire: ## Download SPIRE binaries
	@echo "Downloading SPIRE $(SPIRE_VERSION)..."
	@if [ ! -f /tmp/spire-install/spire-$(SPIRE_VERSION)/bin/spire-agent ]; then \
		rm -rf /tmp/spire-install/spire-$(SPIRE_VERSION); \
		mkdir -p /tmp/spire-install; \
		curl -sL $(SPIRE_URL) | tar xz -C /tmp/spire-install; \
		echo "SPIRE downloaded successfully"; \
	else \
		echo "SPIRE $(SPIRE_VERSION) already downloaded"; \
	fi

create-users: ## Create SPIRE system users
	@echo "Creating SPIRE users..."
	@sudo useradd -r -s /bin/false -d $(SPIRE_DATA_DIR)/server -m spire-server || true
	@sudo useradd -r -s /bin/false -d $(SPIRE_DATA_DIR)/agent -m spire-agent || true

setup-directories: ## Create SPIRE directories
	@echo "Setting up SPIRE directories..."
	@sudo mkdir -p $(SPIRE_INSTALL_DIR)/bin
	@sudo mkdir -p $(SPIRE_INSTALL_DIR)/scripts
	@sudo mkdir -p $(SPIRE_CONFIG_DIR)/server
	@sudo mkdir -p $(SPIRE_CONFIG_DIR)/agent
	@sudo mkdir -p $(SPIRE_DATA_DIR)/server
	@sudo mkdir -p $(SPIRE_DATA_DIR)/agent
	@sudo mkdir -p $(SPIRE_DATA_DIR)/agent/keys
	@sudo mkdir -p /run/spire/sockets

install: install-server install-agent ## Install both SPIRE server and agent

install-server: download-spire create-users setup-directories ## Install SPIRE server
	@echo "Installing SPIRE server..."
	@sudo systemctl stop spire-server 2>/dev/null || true
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-server
	@sudo cp /tmp/spire-install/spire-$(SPIRE_VERSION)/bin/spire-server $(SPIRE_INSTALL_DIR)/bin/
	@sudo chmod +x $(SPIRE_INSTALL_DIR)/bin/spire-server
	@# AIDEV-NOTE: spire-server binary includes all CLI functionality in v1.12.2
	@echo "Installing server configuration for environment: $(SPIRE_ENVIRONMENT)..."
	@echo "Trust domain: $(TRUST_DOMAIN)"
	@sudo cp environments/$(SPIRE_ENVIRONMENT)/server.conf $(SPIRE_CONFIG_DIR)/server/server.conf
	@sudo chown -R spire-server:spire-server $(SPIRE_DATA_DIR)/server
	@sudo chown -R spire-server:spire-server $(SPIRE_CONFIG_DIR)/server
	@echo "Installing systemd service..."
	@sudo cp contrib/systemd/spire-server.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable spire-server
	@echo "SPIRE server installed. Run 'make service-start-server' to start."

install-agent: download-spire create-users setup-directories ## Install SPIRE agent
	@echo "Installing SPIRE agent..."
	@sudo systemctl stop spire-agent 2>/dev/null || true
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-agent
	@sudo cp contrib/systemd/spire-agent.service /etc/systemd/system/spire-agent.service
	@sudo cp /tmp/spire-install/spire-$(SPIRE_VERSION)/bin/spire-agent $(SPIRE_INSTALL_DIR)/bin/
	@sudo chmod +x $(SPIRE_INSTALL_DIR)/bin/spire-agent
	@echo "Installing agent configuration for environment: $(SPIRE_ENVIRONMENT)..."
	@echo "Trust domain: $(TRUST_DOMAIN)"
	@sudo cp environments/$(SPIRE_ENVIRONMENT)/agent.conf $(SPIRE_CONFIG_DIR)/agent/agent.conf
	@sudo chown -R spire-agent:spire-agent $(SPIRE_DATA_DIR)/agent
	@sudo chown -R spire-agent:spire-agent $(SPIRE_CONFIG_DIR)/agent
	@sudo chmod 700 $(SPIRE_DATA_DIR)/agent/keys
	@echo "Installing systemd service..."
	@sudo cp contrib/systemd/spire-agent.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable spire-agent
	@echo "SPIRE agent installed. Bootstrap bundle required before starting."

uninstall: uninstall-server uninstall-agent ## Uninstall both SPIRE server and agent

uninstall-server: ## Uninstall SPIRE server
	@echo "Uninstalling SPIRE server..."
	@sudo systemctl stop spire-server || true
	@sudo systemctl disable spire-server || true
	@sudo rm -f /etc/systemd/system/spire-server.service
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-server
	@sudo systemctl daemon-reload
	@echo "SPIRE server uninstalled."
	@echo "Data preserved in $(SPIRE_DATA_DIR)/server. To fully remove:"
	@echo "  sudo userdel spire-server"
	@echo "  sudo rm -rf $(SPIRE_DATA_DIR)/server $(SPIRE_CONFIG_DIR)/server"

uninstall-agent: ## Uninstall SPIRE agent
	@echo "Uninstalling SPIRE agent..."
	@sudo systemctl stop spire-agent || true
	@sudo systemctl disable spire-agent || true
	@sudo rm -f /etc/systemd/system/spire-agent.service
	@sudo rm -f $(SPIRE_INSTALL_DIR)/bin/spire-agent
	@sudo systemctl daemon-reload
	@echo "SPIRE agent uninstalled."
	@echo "Data preserved in $(SPIRE_DATA_DIR)/agent. To fully remove:"
	@echo "  sudo userdel spire-agent"
	@echo "  sudo rm -rf $(SPIRE_DATA_DIR)/agent $(SPIRE_CONFIG_DIR)/agent"

# Service management - Server
service-start-server: ## Start SPIRE server
	@echo "Starting SPIRE server..."
	sudo systemctl start spire-server

service-stop-server: ## Stop SPIRE server
	@echo "Stopping SPIRE server..."
	sudo systemctl stop spire-server

service-restart-server: ## Restart SPIRE server
	@echo "Restarting SPIRE server..."
	sudo systemctl restart spire-server

service-status-server: ## Check SPIRE server status
	sudo systemctl status spire-server

service-logs-server: ## Follow SPIRE server logs
	sudo journalctl -u spire-server -f

# Service management - Agent
service-start-agent: ## Start SPIRE agent
	@echo "Starting SPIRE agent..."
	sudo systemctl start spire-agent

service-stop-agent: ## Stop SPIRE agent
	@echo "Stopping SPIRE agent..."
	sudo systemctl stop spire-agent

service-restart-agent: ## Restart SPIRE agent
	@echo "Restarting SPIRE agent..."
	sudo systemctl restart spire-agent

service-status-agent: ## Check SPIRE agent status
	sudo systemctl status spire-agent

service-logs-agent: ## Follow SPIRE agent logs
	sudo journalctl -u spire-agent -f

# Combined service management
service-start: service-start-server service-start-agent ## Start both server and agent

service-stop: service-stop-agent service-stop-server ## Stop both agent and server

service-restart: service-restart-server service-restart-agent ## Restart both

service-status: ## Check status of both services
	@echo "=== SPIRE Server ==="
	@sudo systemctl status spire-server --no-pager || true
	@echo ""
	@echo "=== SPIRE Agent ==="
	@sudo systemctl status spire-agent --no-pager || true

service-logs: ## Follow logs for both services
	@echo "Following logs for both services (Ctrl+C to stop)..."
	sudo journalctl -u spire-server -u spire-agent -f

# Bootstrap and setup
bootstrap-agent: ## Bootstrap agent with server bundle
	@echo "Bootstrapping agent with server bundle..."
	@if ! sudo systemctl is-active --quiet spire-server; then \
		echo "Error: SPIRE server is not running. Start it first with 'make service-start-server'"; \
		exit 1; \
	fi
	@echo "Waiting for SPIRE server socket to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if [ -S /run/spire/server.sock ]; then \
			echo "Server socket is ready"; \
			break; \
		fi; \
		echo "Waiting for server socket... ($$i/10)"; \
		sleep 2; \
	done
	@if [ ! -S /run/spire/server.sock ]; then \
		echo "Error: Server socket not available after 20 seconds"; \
		exit 1; \
	fi
	@sudo bash -c '$(SPIRE_INSTALL_DIR)/bin/spire-server bundle show \
		-socketPath /run/spire/server.sock \
		-format pem > /tmp/bootstrap.crt' || \
		(echo "Error: Failed to get bundle from server. Check server logs with 'make service-logs-server'" && exit 1)
	@sudo mv /tmp/bootstrap.crt $(SPIRE_CONFIG_DIR)/agent/bootstrap.crt
	@sudo chown spire-agent:spire-agent $(SPIRE_CONFIG_DIR)/agent/bootstrap.crt
	@sudo chmod 600 $(SPIRE_CONFIG_DIR)/agent/bootstrap.crt
	@echo "Agent bootstrap bundle created."

setup-agent: ## Setup agent with join token
	@echo "Setting up agent with join token..."
	@./scripts/setup-agent.sh

register-agent: ## Register agent with join token (one-time setup)
	@echo "Registering agent..."
	@./scripts/register-agent.sh

register-services: ## Register all Unkey services with SPIRE
	@echo "Registering services with SPIRE..."
	@TRUST_DOMAIN=$(TRUST_DOMAIN) ./scripts/register-services.sh

create-join-token: ## Create join token for agent
	@echo "Creating join token for trust domain: $(TRUST_DOMAIN)..."
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server token generate \
		-socketPath /run/spire/server.sock \
		-spiffeID spiffe://$(TRUST_DOMAIN)/agent/node1


register-metald: ## Register metald service
	@echo "Registering metald in trust domain: $(TRUST_DOMAIN)..."
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server entry create \
		-socketPath /var/lib/spire/server/server.sock \
		-parentID spiffe://$(TRUST_DOMAIN)/agent/node1 \
		-spiffeID spiffe://$(TRUST_DOMAIN)/service/metald \
		-selector unix:user:metald \
		-selector systemd:service:metald

register-billaged: ## Register billaged service
	@echo "Registering billaged in trust domain: $(TRUST_DOMAIN)..."
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server entry create \
		-socketPath /var/lib/spire/server/server.sock \
		-parentID spiffe://$(TRUST_DOMAIN)/agent/node1 \
		-spiffeID spiffe://$(TRUST_DOMAIN)/service/billaged \
		-selector unix:user:billaged \
		-selector systemd:service:billaged

register-builderd: ## Register builderd service
	@echo "Registering builderd in trust domain: $(TRUST_DOMAIN)..."
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server entry create \
		-socketPath /var/lib/spire/server/server.sock \
		-parentID spiffe://$(TRUST_DOMAIN)/agent/node1 \
		-spiffeID spiffe://$(TRUST_DOMAIN)/service/builderd \
		-selector unix:user:builderd \
		-selector systemd:service:builderd

register-assetmanagerd: ## Register assetmanagerd service
	@echo "Registering assetmanagerd in trust domain: $(TRUST_DOMAIN)..."
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server entry create \
		-socketPath /var/lib/spire/server/server.sock \
		-parentID spiffe://$(TRUST_DOMAIN)/agent/node1 \
		-spiffeID spiffe://$(TRUST_DOMAIN)/service/assetmanagerd \
		-selector unix:user:assetmanagerd \
		-selector systemd:service:assetmanagerd

list-entries: ## List all registered entries
	@sudo $(SPIRE_INSTALL_DIR)/bin/spire-server entry show \
		-socketPath /run/spire/server.sock

clean: ## Clean up temporary files
	@rm -rf /tmp/spire-install

clean-all: ## Completely remove SPIRE (data, configs, users)
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
	@sudo rm -rf /run/spire
	@sudo rm -rf /var/run/spire
	@echo "Removing SPIRE users..."
	@sudo userdel -r spire-server 2>/dev/null || true
	@sudo userdel -r spire-agent 2>/dev/null || true
	@echo "SPIRE completely removed."
