#!/bin/bash
# Registers all Unkey services with proper selectors

set -euo pipefail

# Get trust domain from environment or use default
TRUST_DOMAIN=${TRUST_DOMAIN:-development.unkey.cloud}
SPIRE_DIR="/opt/spire"
SOCKET_PATH="/var/lib/spire/server/server.sock"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== SPIRE Service Registration ===${NC}"
echo -e "Trust Domain: ${YELLOW}${TRUST_DOMAIN}${NC}"

# Check if server is running
#if ! systemctl is-active --quiet spire-server; then
#    echo -e "${RED}Error: SPIRE server is not running${NC}"
#    echo "Start it with: sudo systemctl start spire-server"
#    exit 1
#fi

# Wait for server socket
if [ ! -S "$SOCKET_PATH" ]; then
    echo -e "${RED}Error: Server socket not available${NC}"
    exit 1
fi

# Function to register a service
register_service() {
    local service_name=$1
    local service_path=$2
    local service_user=$3
    local parent_id="spiffe://${TRUST_DOMAIN}/agent/node1"
    local spiffe_id="spiffe://${TRUST_DOMAIN}/service/${service_name}"

    echo -e "\n${BLUE}Registering ${service_name}...${NC}"

    # Check if entry already exists
    if sudo ${SPIRE_DIR}/bin/spire-server entry show \
        -socketPath "$SOCKET_PATH" \
        -spiffeID "$spiffe_id" 2>/dev/null | grep -q "SPIFFE ID"; then
        echo -e "${YELLOW}✓ ${service_name} already registered${NC}"
        return 0
    fi

    # Create registration entry
    sudo ${SPIRE_DIR}/bin/spire-server entry create \
        -socketPath "$SOCKET_PATH" \
        -parentID "$parent_id" \
        -spiffeID "$spiffe_id" \
        -selector "unix:path:${service_path}" \
        -selector "unix:user:${service_user}" \
        -x509SVIDTTL 3600 \
        || {
            echo -e "${RED}✗ Failed to register ${service_name}${NC}"
            return 1
        }

    echo -e "${GREEN}✓ ${service_name} registered${NC}"
}

# Register all services
# All services run as their own dedicated user
register_service "metald" "/usr/local/bin/metald" "root"
register_service "billaged" "/usr/local/bin/billaged" "billaged"
register_service "builderd" "/usr/local/bin/builderd" "root"
register_service "assetmanagerd" "/usr/local/bin/assetmanagerd" "root"

# Register the CLI tools for testing
register_service "metald-cli" "/usr/local/bin/metald-cli" "$USER"
register_service "assetmanagerd-cli" "/usr/local/bin/assetmanagerd-cli" "$USER"
register_service "billaged-cli" "/usr/local/bin/billaged-cli" "$USER"
register_service "builderd-cli" "/usr/local/bin/builderd-cli" "$USER"

# List all registered entries
echo -e "\n${YELLOW}=== Registered Services ===${NC}"
sudo ${SPIRE_DIR}/bin/spire-server entry show \
    -socketPath "$SOCKET_PATH" \
    -parentID "spiffe://${TRUST_DOMAIN}/agent/node1" \
    | grep -E "(Entry ID|SPIFFE ID|Selector)" || true

echo -e "\n${GREEN}✓ Service registration complete!${NC}"
echo -e "\n${YELLOW}Next steps:${NC}"
echo "1. Start services with SPIFFE support enabled"
echo "2. Services will automatically receive SVIDs from SPIRE"
echo "3. Monitor logs: sudo journalctl -u spire-agent -f"
