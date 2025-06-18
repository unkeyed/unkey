#!/bin/bash
# AIDEV-NOTE: Preprocesses SPIRE configuration templates with environment variable substitution
# This handles shell-style variable expansion that SPIRE doesn't support natively

set -euo pipefail

# Check if running as root for directory creation
if [[ $EUID -eq 0 ]]; then
    # Create necessary directories
    mkdir -p /run/spire /var/lib/spire/server/{data,keys} /etc/spire/server
    chown -R spire-server:spire-server /run/spire /var/lib/spire/server
    chmod 755 /run/spire
    chmod 700 /var/lib/spire/server/keys
fi

# Configuration template path
TEMPLATE_PATH="${1:-/etc/spire/server/server.conf.template}"
OUTPUT_PATH="${2:-/etc/spire/server/server.conf}"

# Check if template exists
if [[ ! -f "$TEMPLATE_PATH" ]]; then
    echo "ERROR: Configuration template not found at $TEMPLATE_PATH"
    exit 1
fi

# Set defaults for environment variables
export UNKEY_SPIRE_TRUST_DOMAIN="${UNKEY_SPIRE_TRUST_DOMAIN:-dev.unkey.app}"
export UNKEY_SPIRE_LOG_LEVEL="${UNKEY_SPIRE_LOG_LEVEL:-INFO}"
export UNKEY_SPIRE_DB_TYPE="${UNKEY_SPIRE_DB_TYPE:-sqlite3}"
export UNKEY_SPIRE_DB_CONNECTION="${UNKEY_SPIRE_DB_CONNECTION:-/var/lib/spire/server/data/datastore.sqlite3}"

# Process the template
echo "Processing SPIRE configuration template..."
echo "Trust Domain: $UNKEY_SPIRE_TRUST_DOMAIN"
echo "Log Level: $UNKEY_SPIRE_LOG_LEVEL"
echo "Database Type: $UNKEY_SPIRE_DB_TYPE"

# Use envsubst to replace variables
# First, we need to escape any $ that aren't part of our variables
cp "$TEMPLATE_PATH" "$OUTPUT_PATH.tmp"

# Replace our specific variables
sed -i "s|\${UNKEY_SPIRE_TRUST_DOMAIN:-[^}]*}|$UNKEY_SPIRE_TRUST_DOMAIN|g" "$OUTPUT_PATH.tmp"
sed -i "s|\${UNKEY_SPIRE_LOG_LEVEL:-[^}]*}|$UNKEY_SPIRE_LOG_LEVEL|g" "$OUTPUT_PATH.tmp"
sed -i "s|\${UNKEY_SPIRE_DB_TYPE:-[^}]*}|$UNKEY_SPIRE_DB_TYPE|g" "$OUTPUT_PATH.tmp"
sed -i "s|\${UNKEY_SPIRE_DB_CONNECTION:-[^}]*}|$UNKEY_SPIRE_DB_CONNECTION|g" "$OUTPUT_PATH.tmp"

# Move the processed file to the final location
mv "$OUTPUT_PATH.tmp" "$OUTPUT_PATH"

# Set proper permissions
if [[ $EUID -eq 0 ]]; then
    chown spire-server:spire-server "$OUTPUT_PATH"
    chmod 640 "$OUTPUT_PATH"
fi

echo "SPIRE configuration processed successfully at $OUTPUT_PATH"