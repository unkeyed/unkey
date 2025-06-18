#!/bin/bash
# AIDEV-NOTE: Debug script to diagnose SPIFFE/SPIRE connection issues

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== SPIFFE/SPIRE Debug Tool ===${NC}"
echo ""

# Check SPIRE agent status
echo -e "${YELLOW}1. Checking SPIRE agent status...${NC}"
if systemctl is-active --quiet spire-agent; then
    echo -e "${GREEN}✓ SPIRE agent is running${NC}"
    systemctl status spire-agent --no-pager | head -n 5
else
    echo -e "${RED}✗ SPIRE agent is not running${NC}"
    echo "  Start with: sudo systemctl start spire-agent"
fi
echo ""

# Check agent socket
echo -e "${YELLOW}2. Checking SPIRE agent socket...${NC}"
SOCKET_PATH="/run/spire/sockets/agent.sock"
if [ -S "$SOCKET_PATH" ]; then
    echo -e "${GREEN}✓ Socket exists at $SOCKET_PATH${NC}"
    ls -la "$SOCKET_PATH"
    
    # Check socket permissions
    if [ -r "$SOCKET_PATH" ] && [ -w "$SOCKET_PATH" ]; then
        echo -e "${GREEN}✓ Socket is readable/writable by current user${NC}"
    else
        echo -e "${RED}✗ Socket is not accessible by current user${NC}"
        echo "  Current user: $(whoami)"
        echo "  User groups: $(groups)"
        echo "  Try: sudo usermod -a -G spire $(whoami)"
    fi
else
    echo -e "${RED}✗ Socket not found at $SOCKET_PATH${NC}"
    echo "  Check SPIRE agent configuration"
fi
echo ""

# Test SPIRE agent health
echo -e "${YELLOW}3. Testing SPIRE agent health...${NC}"
if command -v /opt/spire/bin/spire-agent &> /dev/null; then
    if timeout 5 /opt/spire/bin/spire-agent healthcheck -socketPath "$SOCKET_PATH" 2>/dev/null; then
        echo -e "${GREEN}✓ SPIRE agent healthcheck passed${NC}"
    else
        echo -e "${RED}✗ SPIRE agent healthcheck failed${NC}"
        echo "  Check agent logs: sudo journalctl -u spire-agent -n 50"
    fi
else
    echo -e "${YELLOW}! spire-agent binary not found, skipping healthcheck${NC}"
fi
echo ""

# Check workload API
echo -e "${YELLOW}4. Testing Workload API connection...${NC}"
if command -v curl &> /dev/null; then
    # Try to connect to the workload API
    if timeout 2 curl -s --unix-socket "$SOCKET_PATH" http://localhost/health &> /dev/null; then
        echo -e "${GREEN}✓ Can connect to Workload API${NC}"
    else
        echo -e "${RED}✗ Cannot connect to Workload API${NC}"
    fi
else
    echo -e "${YELLOW}! curl not available, skipping API test${NC}"
fi
echo ""

# Check current registrations
echo -e "${YELLOW}5. Checking SPIRE registrations...${NC}"
if [ -S "/run/spire/server.sock" ]; then
    echo "Registered entries for this client:"
    sudo /opt/spire/bin/spire-server entry show -socketPath /run/spire/server.sock 2>/dev/null | \
        grep -A5 -B5 "metald-example-client" || echo "  No entries found for metald-example-client"
else
    echo -e "${YELLOW}! SPIRE server socket not found, cannot check registrations${NC}"
fi
echo ""

# Test with Go SPIFFE library
echo -e "${YELLOW}6. Testing Go SPIFFE library...${NC}"
cat > /tmp/test-spiffe.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    fmt.Println("Attempting to connect to SPIRE agent...")
    
    source, err := workloadapi.NewX509Source(ctx,
        workloadapi.WithClientOptions(
            workloadapi.WithAddr("unix:///run/spire/sockets/agent.sock"),
        ),
    )
    if err != nil {
        log.Fatalf("Failed to create X509 source: %v", err)
    }
    defer source.Close()
    
    svid, err := source.GetX509SVID()
    if err != nil {
        log.Fatalf("Failed to get SVID: %v", err)
    }
    
    fmt.Printf("Success! Got SPIFFE ID: %s\n", svid.ID.String())
}
EOF

if command -v go &> /dev/null; then
    echo "Testing Go SPIFFE connection..."
    cd /tmp && go mod init test-spiffe 2>/dev/null || true
    go get github.com/spiffe/go-spiffe/v2@latest 2>/dev/null || true
    if timeout 15 go run test-spiffe.go 2>&1; then
        echo -e "${GREEN}✓ Go SPIFFE library can connect${NC}"
    else
        echo -e "${RED}✗ Go SPIFFE library failed to connect${NC}"
    fi
    rm -f /tmp/test-spiffe.go /tmp/go.mod /tmp/go.sum
else
    echo -e "${YELLOW}! Go not available, skipping library test${NC}"
fi
echo ""

# Environment check
echo -e "${YELLOW}7. Environment check...${NC}"
echo "SPIFFE-related environment variables:"
env | grep -i spiffe || echo "  None found"
echo ""

# Summary
echo -e "${BLUE}=== Summary ===${NC}"
echo ""
echo "If the client fails with 'context deadline exceeded', check:"
echo "1. SPIRE agent is running and healthy"
echo "2. Socket permissions allow access"
echo "3. Client binary is registered with correct selectors"
echo "4. No firewall/SELinux blocking Unix socket access"
echo ""
echo "To see detailed SPIRE agent logs:"
echo "  sudo journalctl -u spire-agent -f"
echo ""
echo "To test manual registration:"
echo "  sudo /opt/spire/bin/spire-server entry create \\"
echo "    -socketPath /run/spire/server.sock \\"
echo "    -parentID spiffe://development.unkey.app/agent/node1 \\"
echo "    -spiffeID spiffe://development.unkey.app/test/manual \\"
echo "    -selector unix:uid:$(id -u)"