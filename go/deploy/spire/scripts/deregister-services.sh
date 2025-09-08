#!/bin/bash
# Deregisters all Unkey services

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

echo -e "${GREEN}=== SPIRE Service Deregistration ===${NC}"
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

# Function to deregister a service
deregister_service() {
    local service_name=$1
    local spiffe_id="spiffe://${TRUST_DOMAIN}/service/${service_name}"

    echo -e "\n${BLUE}Deregistering ${service_name}...${NC}"
    echo -e "\n${BLUE}spiffeid: ${spiffe_id}${NC}"
    # Find entry ID for the service
    local entry_id=$(sudo ${SPIRE_DIR}/bin/spire-server entry show \
        -socketPath "$SOCKET_PATH" \
        -spiffeID "$spiffe_id" 2>/dev/null | grep "Entry ID" | awk '{print $NF}')

    if [ -z "$entry_id" ]; then
        echo -e "${YELLOW}✓ ${service_name} not registered${NC}"
        return 0
    fi
    echo -e "${GREEN}✓ ${service_name} registered${NC}"

    # Delete the entry
    sudo ${SPIRE_DIR}/bin/spire-server entry delete \
        -socketPath "$SOCKET_PATH" \
        -entryID "$entry_id" \
        || {
            echo -e "${RED}✗ Failed to deregister ${service_name}${NC}"
            return 1
        }

    echo -e "${GREEN}✓ ${service_name} deregistered${NC}"
}

# Deregister all services
deregister_service "metald"
deregister_service "billaged"
deregister_service "builderd"
deregister_service "assetmanagerd"
deregister_service "metald-cli"
deregister_service "assetmanagerd-cli"
deregister_service "billaged-cli"
deregister_service "builderd-cli"
deregister_service "metald-client"

# List remaining registered entries
echo -e "\n${YELLOW}=== Remaining Registered Services ===${NC}"
sudo ${SPIRE_DIR}/bin/spire-server entry show \
    -socketPath "$SOCKET_PATH" \
    -parentID "spiffe://${TRUST_DOMAIN}/agent/node1" \
    | grep -E "(Entry ID|SPIFFE ID|Selector)" || echo -e "${GREEN}No services remaining${NC}"

echo -e "\n${GREEN}✓ Service deregistration complete!${NC}"
