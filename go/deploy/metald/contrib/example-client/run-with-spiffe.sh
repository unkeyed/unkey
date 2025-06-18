#!/bin/bash
# AIDEV-NOTE: Helper script to run the example client with SPIFFE/SPIRE

set -euo pipefail

# Default values
METALD_ADDR="${UNKEY_METALD_ADDR:-https://localhost:8080}"
CUSTOMER_ID="${UNKEY_METALD_CUSTOMER_ID:-example-customer}"
SPIFFE_SOCKET="${UNKEY_METALD_SPIFFE_SOCKET:-/run/spire/sockets/agent.sock}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if SPIRE agent is available
check_spire() {
    echo -e "${YELLOW}Checking SPIFFE/SPIRE setup...${NC}"
    
    if [ ! -S "$SPIFFE_SOCKET" ]; then
        echo -e "${RED}Error: SPIFFE socket not found at $SPIFFE_SOCKET${NC}"
        echo "Please ensure SPIRE agent is running or set UNKEY_METALD_SPIFFE_SOCKET"
        exit 1
    fi
    
    # Check if we can connect to the socket
    if ! timeout 2 nc -U "$SPIFFE_SOCKET" < /dev/null; then
        echo -e "${RED}Error: Cannot connect to SPIFFE socket${NC}"
        echo "Please check SPIRE agent status: systemctl status spire-agent"
        exit 1
    fi
    
    echo -e "${GREEN}SPIFFE/SPIRE is available${NC}"
}

# Build the client if needed
build_client() {
    if [ ! -f "./metald-client" ] || [ "main.go" -nt "./metald-client" ]; then
        echo -e "${YELLOW}Building metald-client...${NC}"
        make build
    fi
}

# Main execution
main() {
    echo "=== Metald Example Client with SPIFFE/SPIRE ==="
    echo "Server: $METALD_ADDR"
    echo "Customer: $CUSTOMER_ID"
    echo ""
    
    # Check prerequisites
    check_spire
    build_client
    
    # Run the client
    echo -e "${YELLOW}Running client with SPIFFE...${NC}"
    ./metald-client \
        --addr "$METALD_ADDR" \
        --customer "$CUSTOMER_ID" \
        --tls-mode spiffe \
        --spiffe-socket "$SPIFFE_SOCKET" \
        "$@"
}

main "$@"