#!/bin/bash
# This wrapper handles automatic joining with server using join tokens
# Environment variables are passed from systemd service

set -euo pipefail

# Configuration from environment variables (set by systemd)
TRUST_DOMAIN=${UNKEY_SPIRE_TRUST_DOMAIN:-development.unkey.cloud}
LOG_LEVEL=${UNKEY_SPIRE_LOG_LEVEL:-INFO}
SERVER_URL=${UNKEY_SPIRE_SERVER_URL:-https://localhost:8085}
JOIN_TOKEN=${UNKEY_SPIRE_JOIN_TOKEN:-}

SPIRE_AGENT="/opt/spire/bin/spire-agent"
CONFIG_FILE="/etc/spire/agent/agent.conf"
LOG_FILE="/var/log/spire-agent.log"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "Starting SPIRE Agent wrapper"
log "Trust Domain: $TRUST_DOMAIN"
log "Log Level: $LOG_LEVEL"
log "Server URL: $SERVER_URL"

# Check if spire-agent binary exists
if [ ! -x "$SPIRE_AGENT" ]; then
    log "ERROR: SPIRE agent binary not found at $SPIRE_AGENT"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    log "ERROR: SPIRE agent config not found at $CONFIG_FILE"
    exit 1
fi

# Build agent command
AGENT_CMD=("$SPIRE_AGENT" "run" "-config" "$CONFIG_FILE")

# Add join token if provided
if [ -n "$JOIN_TOKEN" ]; then
    log "Using join token for authentication"
    AGENT_CMD+=("-joinToken" "$JOIN_TOKEN")
else
    log "No join token provided - using bootstrap bundle"
fi

log "Starting SPIRE agent: ${AGENT_CMD[*]}"

# Execute the agent
exec "${AGENT_CMD[@]}"
