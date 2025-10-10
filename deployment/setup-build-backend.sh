#!/bin/bash
set -e

BACKEND="${1:-}"

if [ "$BACKEND" != "depot" ] && [ "$BACKEND" != "docker" ]; then
  echo "Usage: $0 [depot|docker]"
  echo ""
  echo "Examples:"
  echo "  $0 depot   # Switch to Depot builds"
  echo "  $0 docker  # Switch to local Docker builds"
  exit 1
fi

if [ "$BACKEND" = "depot" ]; then
  # Try to read token from config file first
  CONFIG_FILE="./config/depot.json"

  if [ -f "$CONFIG_FILE" ]; then
    echo "Reading token from $CONFIG_FILE..."
    DEPOT_TOKEN=$(grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | sed 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

    if [ -z "$DEPOT_TOKEN" ]; then
      echo "Warning: Could not parse token from $CONFIG_FILE"
    else
      echo "✓ Token loaded from config file"
    fi
  fi

  # If still no token, check environment variable
  if [ -z "$DEPOT_TOKEN" ]; then
    if [ -n "${DEPOT_TOKEN:-}" ]; then
      DEPOT_TOKEN="${DEPOT_TOKEN}"
      echo "✓ Using token from DEPOT_TOKEN environment variable"
    fi
  fi

  # If still no token, prompt user
  if [ -z "$DEPOT_TOKEN" ]; then
    echo "Enter your Depot token:"
    read -r -s DEPOT_TOKEN
    echo ""

    if [ -z "$DEPOT_TOKEN" ]; then
      echo "Error: No token provided"
      exit 1
    fi
  fi

  if [ ${#DEPOT_TOKEN} -lt 20 ]; then
    echo "Error: Token appears to be invalid (too short)"
    exit 1
  fi

  if echo "$DEPOT_TOKEN" | docker login registry.depot.dev -u x-token --password-stdin >/dev/null 2>&1; then
    echo "✓ Logged into depot registry"
  else
    echo ""
    echo "Error: Docker login failed. Invalid token?"
    exit 1
  fi

  # Write to .env file
  cat >.env <<EOF
UNKEY_BUILD_BACKEND=depot
DEPOT_TOKEN=$DEPOT_TOKEN
EOF
  echo "✓ Updated .env file with depot backend"
else
  echo "Logging out from depot registry..."
  docker logout registry.depot.dev 2>/dev/null || true
  echo "✓ Switched to local Docker"

  # Write to .env file
  cat >.env <<EOF
UNKEY_BUILD_BACKEND=docker
EOF
  echo "✓ Updated .env file with docker backend"
fi

echo "✓ Done! Build backend set to: $BACKEND"
