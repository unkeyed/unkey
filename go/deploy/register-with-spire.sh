#!/bin/bash
# register-with-spire.sh - Shared SPIRE registration script

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <service-name> [dev]"
    echo "Example: $0 test-metald-client"
    echo "Example: $0 test-metald-client dev"
    exit 1
fi

SERVICE_NAME="$1"
MODE="${2:-prod}"

SPIFFE_ID="spiffe://development.unkey.app/client/${SERVICE_NAME}"
PARENT_ID="spiffe://development.unkey.app/agent/$(hostname)"
X509_TTL=3600  # 1 hour

# Get current user and binary path
USERNAME=$(whoami)
BINARY_PATH=$(realpath "./build/${SERVICE_NAME}")

if [ "$MODE" == "dev" ]; then
    echo "Registering ${SERVICE_NAME} in development mode (UID selector only)..."
    sudo /opt/spire/bin/spire-server entry create \
        -socketPath /var/lib/spire/server/server.sock \
        -spiffeID "${SPIFFE_ID}" \
        -parentID "${PARENT_ID}" \
        -selector "unix:uid:$(id -u)" \
        -x509SVIDTTL ${X509_TTL} \
        -admin
else
    echo "Registering ${SERVICE_NAME} in production mode (user + path selectors)..."
    sudo /opt/spire/bin/spire-server entry create \
        -socketPath /var/lib/spire/server/server.sock \
        -spiffeID "${SPIFFE_ID}" \
        -parentID "${PARENT_ID}" \
        -selector "unix:user:${USERNAME}" \
        -selector "unix:path:${BINARY_PATH}" \
        -x509SVIDTTL ${X509_TTL} \
        -admin
fi

echo "Successfully registered ${SERVICE_NAME} with SPIRE"
echo "SPIFFE ID: ${SPIFFE_ID}"