#!/bin/bash
# Quick test to verify SPIFFE connectivity

echo "Testing SPIFFE/SPIRE connection..."

# Check if agent socket exists
SOCKET="/run/spire/sockets/agent.sock"
if [ ! -S "$SOCKET" ]; then
    echo "ERROR: SPIRE agent socket not found at $SOCKET"
    echo "Is SPIRE agent running? Check: systemctl status spire-agent"
    exit 1
fi

echo "✓ Socket found at $SOCKET"

# Check permissions
if [ ! -r "$SOCKET" ]; then
    echo "ERROR: Cannot read socket. Current user: $(whoami)"
    echo "Groups: $(groups)"
    echo "Socket info:"
    ls -la "$SOCKET"
    echo ""
    echo "To fix, try:"
    echo "  sudo usermod -a -G spire $(whoami)"
    echo "  Then logout and login again"
    exit 1
fi

echo "✓ Socket is readable"

# Try healthcheck if available
if command -v /opt/spire/bin/spire-agent &> /dev/null; then
    echo "Running agent healthcheck..."
    if /opt/spire/bin/spire-agent healthcheck -socketPath "$SOCKET"; then
        echo "✓ Agent healthcheck passed"
    else
        echo "✗ Agent healthcheck failed"
        echo "Check logs: sudo journalctl -u spire-agent -n 20"
    fi
fi

echo ""
echo "Socket permissions:"
ls -la "$SOCKET"
echo ""
echo "Your user info:"
echo "  User: $(whoami)"
echo "  UID: $(id -u)"
echo "  Groups: $(groups)"

# Test if we're registered
echo ""
echo "Checking if you have any SPIRE registrations matching your UID..."
if [ -S "/run/spire/server.sock" ]; then
    sudo /opt/spire/bin/spire-server entry show -socketPath /run/spire/server.sock 2>/dev/null | \
        grep -B5 -A5 "unix:uid:$(id -u)" | head -20 || echo "No entries found for UID $(id -u)"
fi