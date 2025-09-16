#!/bin/bash
# For development: creates a long-lived token that enables auto-joining on startup
# For production: creates shorter-lived tokens with node attestation

set -euo pipefail

# Get trust domain from environment or use default
TRUST_DOMAIN=${UNKEY_SPIRE_TRUST_DOMAIN:-development.unkey.cloud}
ENVIRONMENT=${SPIRE_ENVIRONMENT:-development}
SPIRE_DIR="/opt/spire"
AGENT_SERVICE_DIR="/etc/systemd/system/spire-agent.service.d"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== SPIRE Agent Auto-Join Setup ===${NC}"
echo -e "Environment: ${YELLOW}${ENVIRONMENT}${NC}"
echo -e "Trust Domain: ${YELLOW}${TRUST_DOMAIN}${NC}"

# Check if server is running
if ! systemctl is-active --quiet spire-server; then
    echo -e "${RED}Error: SPIRE server is not running${NC}"
    echo "Start it with: sudo systemctl start spire-server"
    exit 1
fi

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

# Check if auto-join is already configured
if [ -f "$AGENT_SERVICE_DIR/auto-join.conf" ]; then
    echo -e "${YELLOW}Auto-join already configured${NC}"
    echo "Checking if agent is running..."
    
    if systemctl is-active --quiet spire-agent; then
        echo -e "${GREEN}✓ Agent is running with auto-join${NC}"
        exit 0
    fi
fi

# Generate join token
echo "Generating join token..."

# For development, create very long-lived token (1 year)
# For production, shorter-lived tokens are recommended
if [ "$ENVIRONMENT" = "development" ]; then
    TTL="31536000"  # 1 year in seconds
    echo -e "${YELLOW}Creating long-lived token for development (1 year)${NC}"
else
    TTL="3600"  # 1 hour in seconds
    echo -e "${YELLOW}Creating token for ${ENVIRONMENT} (1 hour)${NC}"
fi

JOIN_TOKEN=$(sudo ${SPIRE_DIR}/bin/spire-server token generate \
    -socketPath /var/lib/spire/server/server.sock \
    -spiffeID spiffe://${TRUST_DOMAIN}/agent/node1 \
    -ttl ${TTL} \
    | grep "Token:" | cut -d' ' -f2)

if [ -z "$JOIN_TOKEN" ]; then
    echo -e "${RED}Error: Failed to generate join token${NC}"
    exit 1
fi

echo -e "${GREEN}Join token generated${NC}"

# Configure systemd for auto-join
echo "Setting up auto-join configuration..."
sudo mkdir -p "$AGENT_SERVICE_DIR"

# Update auto-join environment configuration with the token
cat <<EOF | sudo tee "$AGENT_SERVICE_DIR/auto-join.conf" > /dev/null
[Service]
# This file provides the join token for automatic agent registration
Environment="UNKEY_SPIRE_JOIN_TOKEN=${JOIN_TOKEN}"
Environment="UNKEY_SPIRE_TRUST_DOMAIN=${TRUST_DOMAIN}"
EOF

# Reload systemd and start agent
sudo systemctl daemon-reload

# Enable auto-start
sudo systemctl enable spire-agent

# Start agent
echo "Starting SPIRE agent with auto-join..."
sudo systemctl restart spire-agent

# Wait for agent to start
echo "Waiting for agent to initialize..."
for i in {1..15}; do
    if systemctl is-active --quiet spire-agent && \
       [ -S /var/lib/spire/agent/agent.sock ]; then
        echo -e "${GREEN}✓ Agent started and socket ready${NC}"
        break
    fi
    echo "Waiting... ($i/15)"
    sleep 2
done

# Verify agent is working
if systemctl is-active --quiet spire-agent; then
    echo -e "${GREEN}✓ SPIRE agent auto-join configured successfully${NC}"
    
    # Test agent health
    echo -e "\n${YELLOW}Agent Health Check:${NC}"
    curl -sv http://localhost:9990/live && echo || echo "Health check endpoint not ready yet"
    
    # Show token expiry warning for non-development environments
    if [ "$ENVIRONMENT" != "development" ]; then
        echo -e "\n${YELLOW}⚠ Token expires in ${TTL}${NC}"
        echo -e "For production, consider using node attestation instead"
    fi

    echo -e "\n${YELLOW}Auto-join configured! Agent will now:${NC}"
    echo "✓ Start automatically on boot"
    echo "✓ Join the SPIRE server automatically"
    echo "✓ Re-join after restarts (until token expires)"
    
    echo -e "\n${YELLOW}Next steps:${NC}"
    echo "1. Register services: make register-services"
    echo "2. View agent logs: sudo journalctl -u spire-agent -f"
    echo "3. Test agent: sudo journalctl -u spire-agent -n 20"
else
    echo -e "${RED}✗ Failed to start SPIRE agent${NC}"
    echo "Check logs with: sudo journalctl -u spire-agent -n 50"
    exit 1
fi
