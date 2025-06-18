#!/bin/bash
# AIDEV-NOTE: Automated agent registration with systemd integration
# This script handles the complete agent registration workflow

set -euo pipefail

# Get trust domain from environment or use default
TRUST_DOMAIN=${TRUST_DOMAIN:-development.unkey.app}
SPIRE_DIR="/opt/spire"
AGENT_SERVICE_DIR="/etc/systemd/system/spire-agent.service.d"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== SPIRE Agent Registration ===${NC}"
echo -e "Trust Domain: ${YELLOW}${TRUST_DOMAIN}${NC}"

# Check if server is running
#if ! systemctl is-active --quiet spire-server; then
#    echo -e "${RED}Error: SPIRE server is not running${NC}"
#    echo "Start it with: sudo systemctl start spire-server"
#    exit 1
#fi

# Wait for server socket
echo "Waiting for SPIRE server socket..."
for i in {1..10}; do
    if [ -S /var/lib/spire/server/server.sock ]; then
        echo -e "${GREEN}Server socket ready${NC}"
        break
    fi
    echo "Waiting... ($i/10)"
    sleep 2
done

if [ ! -S /var/lib/spire/server/server.sock ]; then
    echo -e "${RED}Error: Server socket not available after 20 seconds${NC}"
    exit 1
fi

# Check if agent is already registered
if systemctl is-active --quiet spire-agent; then
    echo -e "${YELLOW}Warning: Agent appears to be already running${NC}"
    echo "Checking registration status..."

    # Try to get agent SVID to verify registration
    if sudo ${SPIRE_DIR}/bin/spire-agent api fetch x509 -socketPath /var/lib/spire/agent/agent.sock 2>/dev/null | grep -q "SPIFFE ID"; then
        echo -e "${GREEN}✓ Agent is already registered and running${NC}"
        exit 0
    fi

    echo "Agent is running but not properly registered. Stopping..."
    sudo systemctl stop spire-agent
fi

# Generate join token
echo "Generating join token..."
JOIN_TOKEN=$(sudo ${SPIRE_DIR}/bin/spire-server token generate \
    -socketPath /var/lib/spire/server/server.sock \
    -spiffeID spiffe://${TRUST_DOMAIN}/agent/node1 \
    | grep "Token:" | cut -d' ' -f2)

if [ -z "$JOIN_TOKEN" ]; then
    echo -e "${RED}Error: Failed to generate join token${NC}"
    exit 1
fi

echo -e "${GREEN}Join token generated${NC}"

# Create systemd drop-in for join token
echo "Configuring agent with join token..."
sudo mkdir -p "$AGENT_SERVICE_DIR"
cat <<EOF | sudo tee "$AGENT_SERVICE_DIR/join-token.conf" > /dev/null
[Service]
Environment="SPIRE_AGENT_JOIN_TOKEN=${JOIN_TOKEN}"
EOF

# Update agent service to use join token on first start
cat <<'EOF' | sudo tee "$AGENT_SERVICE_DIR/auto-register.conf" > /dev/null
[Service]
# AIDEV-NOTE: Auto-registration support
# The join token is passed as a command line argument if the env var is set
ExecStart=
ExecStart=/bin/bash -c '\
    if [ -n "${SPIRE_AGENT_JOIN_TOKEN}" ]; then \
        exec /opt/spire/bin/spire-agent run -config /etc/spire/agent/agent.conf -joinToken "${SPIRE_AGENT_JOIN_TOKEN}"; \
    else \
        exec /opt/spire/bin/spire-agent run -config /etc/spire/agent/agent.conf; \
    fi'
EOF

# Reload systemd
sudo systemctl daemon-reload

# Start agent
echo "Starting SPIRE agent..."
sudo systemctl start spire-agent

# Wait for agent to start
echo "Waiting for agent to initialize..."
for i in {1..10}; do
    if systemctl is-active --quiet spire-agent && \
       [ -S /var/lib/spire/agent/agent.sock ]; then
        echo -e "${GREEN}✓ Agent started${NC}"
        break
    fi
    echo "Waiting... ($i/10)"
    sleep 2
done

# Verify registration
if systemctl is-active --quiet spire-agent; then
    echo -e "${GREEN}✓ SPIRE agent started successfully${NC}"

    # Remove join token after successful registration
    echo "Cleaning up join token..."
    sudo rm -f "$AGENT_SERVICE_DIR/join-token.conf"
    sudo systemctl daemon-reload

    echo -e "${GREEN}✓ Agent registration complete!${NC}"

    # Show agent info
    echo -e "\n${YELLOW}Agent Health Check:${NC}"
    curl -sv http://localhost:8084/live && echo || echo "Health check endpoint not ready yet"

    echo -e "\n${YELLOW}Next steps:${NC}"
    echo "1. Register services: make register-services"
    echo "2. View agent logs: sudo journalctl -u spire-agent -f"
else
    echo -e "${RED}✗ Failed to start SPIRE agent${NC}"
    echo "Check logs with: sudo journalctl -u spire-agent -n 50"
    exit 1
fi
