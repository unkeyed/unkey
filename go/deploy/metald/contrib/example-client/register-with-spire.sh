#!/bin/bash
# AIDEV-NOTE: Script to register the metald example client with SPIRE
# This allows the client to obtain a SPIFFE identity for mTLS communication

set -euo pipefail

# Configuration
TRUST_DOMAIN="${TRUST_DOMAIN:-development.unkey.app}"
SPIRE_SERVER_SOCKET="${SPIRE_SERVER_SOCKET:-/run/spire/server.sock}"
SPIRE_BIN="${SPIRE_BIN:-/opt/spire/bin/spire-server}"
CLIENT_NAME="metald-example-client"
CLIENT_USER="${USER}"  # Current user running the client

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    
    # Check if SPIRE server is available
    if [ ! -S "$SPIRE_SERVER_SOCKET" ]; then
        echo -e "${RED}Error: SPIRE server socket not found at $SPIRE_SERVER_SOCKET${NC}"
        echo "Please ensure SPIRE server is running or set SPIRE_SERVER_SOCKET"
        exit 1
    fi
    
    # Check if spire-server binary exists
    if [ ! -x "$SPIRE_BIN" ]; then
        echo -e "${RED}Error: SPIRE server binary not found at $SPIRE_BIN${NC}"
        echo "Please install SPIRE or set SPIRE_BIN to the correct path"
        exit 1
    fi
    
    echo -e "${GREEN}Prerequisites OK${NC}"
}

# Get the parent agent ID
get_agent_id() {
    echo -e "${YELLOW}Finding SPIRE agent...${NC}"
    
    # Try to find the agent entry - look for the actual SPIFFE ID field
    AGENT_ID=$(sudo "$SPIRE_BIN" entry show \
        -socketPath "$SPIRE_SERVER_SOCKET" \
        -spiffeID "spiffe://${TRUST_DOMAIN}/agent/node1" \
        2>/dev/null | grep -E "^\s*SPIFFE ID\s*:" | sed 's/.*SPIFFE ID\s*:\s*//' | tr -d ' ' || true)
    
    # If that didn't work, try finding any agent entry
    if [ -z "$AGENT_ID" ]; then
        AGENT_ID=$(sudo "$SPIRE_BIN" entry show \
            -socketPath "$SPIRE_SERVER_SOCKET" \
            2>/dev/null | grep -E "spiffe://${TRUST_DOMAIN}/agent" | head -1 | sed 's/.*\(spiffe:\/\/[^ ]*\).*/\1/' || true)
    fi
    
    # Default to expected agent ID if still not found
    if [ -z "$AGENT_ID" ]; then
        AGENT_ID="spiffe://${TRUST_DOMAIN}/agent/node1"
        echo -e "${YELLOW}Warning: Could not find agent entry, using default: $AGENT_ID${NC}"
    else
        echo -e "${GREEN}Found agent: $AGENT_ID${NC}"
    fi
}

# Register the client workload
register_client() {
    echo -e "${YELLOW}Registering $CLIENT_NAME with SPIRE...${NC}"
    
    # Build the full path to the client binary
    CLIENT_PATH="$(pwd)/metald-client"
    
    # Create the entry
    echo -e "${BLUE}Creating SPIRE entry...${NC}"
    echo "  SPIFFE ID: spiffe://${TRUST_DOMAIN}/client/${CLIENT_NAME}"
    echo "  Parent ID: $AGENT_ID"
    echo "  Selector: unix:user:${CLIENT_USER}"
    echo "  Selector: unix:path:${CLIENT_PATH}"
    
    # Check if entry already exists
    EXISTING=$(sudo "$SPIRE_BIN" entry show \
        -socketPath "$SPIRE_SERVER_SOCKET" \
        -spiffeID "spiffe://${TRUST_DOMAIN}/client/${CLIENT_NAME}" \
        2>/dev/null || true)
    
    if [ -n "$EXISTING" ]; then
        echo -e "${YELLOW}Entry already exists. Updating...${NC}"
        # Delete existing entry first
        ENTRY_ID=$(echo "$EXISTING" | grep "Entry ID" | awk '{print $3}')
        sudo "$SPIRE_BIN" entry delete \
            -socketPath "$SPIRE_SERVER_SOCKET" \
            -entryID "$ENTRY_ID"
    fi
    
    # Create new entry
    sudo "$SPIRE_BIN" entry create \
        -socketPath "$SPIRE_SERVER_SOCKET" \
        -parentID "$AGENT_ID" \
        -spiffeID "spiffe://${TRUST_DOMAIN}/client/${CLIENT_NAME}" \
        -selector "unix:user:${CLIENT_USER}" \
        -selector "unix:path:${CLIENT_PATH}" \
        -x509SVIDTTL 3600 \
        -admin
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Successfully registered $CLIENT_NAME${NC}"
    else
        echo -e "${RED}Failed to register $CLIENT_NAME${NC}"
        exit 1
    fi
}

# Show registration info
show_info() {
    echo ""
    echo -e "${GREEN}=== Registration Complete ===${NC}"
    echo ""
    echo "The example client has been registered with SPIRE."
    echo ""
    echo "SPIFFE ID: spiffe://${TRUST_DOMAIN}/client/${CLIENT_NAME}"
    echo ""
    echo "The client will be identified by:"
    echo "  - Running as user: ${CLIENT_USER}"
    echo "  - Binary path: $(pwd)/metald-client"
    echo ""
    echo "To use the client with SPIFFE:"
    echo "  1. Build the client: make build"
    echo "  2. Run with SPIFFE: ./run-with-spiffe.sh"
    echo ""
    echo "Example:"
    echo "  ./metald-client --tls-mode spiffe --action list"
    echo ""
}

# Alternative registration for development
register_dev_client() {
    echo -e "${YELLOW}Registering development client (any user/path)...${NC}"
    
    # This registration is less secure but useful for development
    # It allows any process to get the client identity
    sudo "$SPIRE_BIN" entry create \
        -socketPath "$SPIRE_SERVER_SOCKET" \
        -parentID "$AGENT_ID" \
        -spiffeID "spiffe://${TRUST_DOMAIN}/client/${CLIENT_NAME}-dev" \
        -selector "unix:uid:${UID}" \
        -x509SVIDTTL 3600
    
    echo -e "${GREEN}Development client registered${NC}"
    echo "SPIFFE ID: spiffe://${TRUST_DOMAIN}/client/${CLIENT_NAME}-dev"
    echo "This entry matches any process running as UID ${UID}"
}

# Main execution
main() {
    echo -e "${BLUE}=== SPIRE Registration for Metald Example Client ===${NC}"
    echo "Trust Domain: $TRUST_DOMAIN"
    echo ""
    
    check_prerequisites
    get_agent_id
    
    # Check if running as root
    if [ "$EUID" -ne 0 ]; then
        echo -e "${YELLOW}Note: This script needs sudo access to register with SPIRE${NC}"
    fi
    
    # Register based on argument
    case "${1:-prod}" in
        dev)
            register_dev_client
            ;;
        *)
            register_client
            show_info
            ;;
    esac
}

# Show usage
if [ "${1:-}" == "--help" ] || [ "${1:-}" == "-h" ]; then
    echo "Usage: $0 [prod|dev]"
    echo ""
    echo "  prod (default) - Register with strict selectors (user + path)"
    echo "  dev           - Register with relaxed selectors (UID only)"
    echo ""
    echo "Environment variables:"
    echo "  TRUST_DOMAIN         - SPIFFE trust domain (default: development.unkey.app)"
    echo "  SPIRE_SERVER_SOCKET  - Path to SPIRE server socket (default: /run/spire/server.sock)"
    echo "  SPIRE_BIN           - Path to spire-server binary (default: /opt/spire/bin/spire-server)"
    exit 0
fi

main "$@"