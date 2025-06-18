#!/bin/bash
# AIDEV-NOTE: Service unregistration for SPIRE
# Removes all Unkey services from SPIRE registration

set -euo pipefail

# Get trust domain from environment or use default
TRUST_DOMAIN=${TRUST_DOMAIN:-development.unkey.app}
SPIRE_DIR="/opt/spire"
SOCKET_PATH="/run/spire/server.sock"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== SPIRE Service Unregistration ===${NC}"
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

# Function to unregister a service
unregister_service() {
    local service_name=$1
    local spiffe_id="spiffe://${TRUST_DOMAIN}/service/${service_name}"
    
    echo -e "\n${BLUE}Unregistering ${service_name}...${NC}"
    
    # Get entry ID for the service
    local entry_id=$(sudo ${SPIRE_DIR}/bin/spire-server entry show \
        -socketPath "$SOCKET_PATH" \
        -spiffeID "$spiffe_id" 2>/dev/null | grep "Entry ID" | awk '{print $4}')
    
    if [ -z "$entry_id" ]; then
        echo -e "${YELLOW}✓ ${service_name} not registered (nothing to do)${NC}"
        return 0
    fi
    
    # Delete the entry
    sudo ${SPIRE_DIR}/bin/spire-server entry delete \
        -socketPath "$SOCKET_PATH" \
        -entryID "$entry_id" \
        || {
            echo -e "${RED}✗ Failed to unregister ${service_name}${NC}"
            return 1
        }
    
    echo -e "${GREEN}✓ ${service_name} unregistered (Entry ID: ${entry_id})${NC}"
}

# Unregister all services
# AIDEV-NOTE: This matches the services registered in register-services.sh
unregister_service "metald"
unregister_service "billaged"
unregister_service "builderd"
unregister_service "assetmanagerd"

# List remaining registered entries
echo -e "\n${YELLOW}=== Remaining Registered Entries ===${NC}"
remaining_entries=$(sudo ${SPIRE_DIR}/bin/spire-server entry show \
    -socketPath "$SOCKET_PATH" \
    -parentID "spiffe://${TRUST_DOMAIN}/agent/node1" 2>/dev/null | grep -E "(Entry ID|SPIFFE ID)" || true)

if [ -z "$remaining_entries" ]; then
    echo -e "${GREEN}No service entries remaining${NC}"
else
    echo "$remaining_entries"
fi

echo -e "\n${GREEN}✓ Service unregistration complete!${NC}"
echo -e "\n${YELLOW}Note:${NC}"
echo "Services will no longer be able to obtain SVIDs from SPIRE"
echo "Restart services without SPIFFE support or re-register them"