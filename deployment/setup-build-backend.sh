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
  CONFIG_FILE="./config/depot.json"

  # Try to read from config file first
  if [ -f "$CONFIG_FILE" ]; then
    echo "Reading configuration from $CONFIG_FILE..."

    DEPOT_TOKEN=$(grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | sed 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    S3_URL=$(grep -o '"s3_url"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | sed 's/.*"s3_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    S3_ACCESS_KEY_ID=$(grep -o '"s3_access_key_id"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | sed 's/.*"s3_access_key_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    S3_ACCESS_KEY_SECRET=$(grep -o '"s3_access_key_secret"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | sed 's/.*"s3_access_key_secret"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

    [ -n "$DEPOT_TOKEN" ] && echo "✓ Token loaded from config file"
    [ -n "$S3_URL" ] && echo "✓ S3 URL loaded from config file"
    [ -n "$S3_ACCESS_KEY_ID" ] && echo "✓ S3 Access Key ID loaded from config file"
    [ -n "$S3_ACCESS_KEY_SECRET" ] && echo "✓ S3 Access Key Secret loaded from config file"
  fi

  # Fall back to environment variables if not in config file
  DEPOT_TOKEN="${DEPOT_TOKEN:-${DEPOT_TOKEN:-}}"
  S3_URL="${S3_URL:-${UNKEY_BUILD_S3_URL:-}}"
  S3_ACCESS_KEY_ID="${S3_ACCESS_KEY_ID:-${UNKEY_BUILD_S3_ACCESS_KEY_ID:-}}"
  S3_ACCESS_KEY_SECRET="${S3_ACCESS_KEY_SECRET:-${UNKEY_BUILD_S3_ACCESS_KEY_SECRET:-}}"

  # Prompt for missing values
  if [ -z "$DEPOT_TOKEN" ]; then
    echo "Enter your Depot token:"
    read -r -s DEPOT_TOKEN
    echo ""
    if [ -z "$DEPOT_TOKEN" ]; then
      echo "Error: No token provided"
      exit 1
    fi
  fi

  # Validate depot token format
  if [[ ! "$DEPOT_TOKEN" =~ ^depot_org_ ]]; then
    echo "Error: Invalid Depot token format. Token must start with 'depot_org_'"
    exit 1
  fi

  if echo "$DEPOT_TOKEN" | docker login registry.depot.dev -u x-token --password-stdin >/dev/null 2>&1; then
    echo "✓ Logged into depot registry"
  else
    echo ""
    echo "Error: Docker login failed. Invalid token?"
    exit 1
  fi

  # Get S3 credentials
  echo ""
  echo "S3 Configuration for Depot:"
  echo "----------------------------"

  if [ -z "$S3_URL" ]; then
    echo "Enter S3 URL (e.g., https://aaar2.cloudflarestorage.com):"
    read -r S3_URL
    if [ -z "$S3_URL" ]; then
      echo "Error: S3 URL is required"
      exit 1
    fi
  fi

  if [ -z "$S3_ACCESS_KEY_ID" ]; then
    echo "Enter S3 Access Key ID:"
    read -r S3_ACCESS_KEY_ID
    if [ -z "$S3_ACCESS_KEY_ID" ]; then
      echo "Error: S3 Access Key ID is required"
      exit 1
    fi
  fi

  if [ -z "$S3_ACCESS_KEY_SECRET" ]; then
    echo "Enter S3 Access Key Secret:"
    read -r -s S3_ACCESS_KEY_SECRET
    echo ""
    if [ -z "$S3_ACCESS_KEY_SECRET" ]; then
      echo "Error: S3 Access Key Secret is required"
      exit 1
    fi
  fi

  # Write to .env file
  cat >.env <<EOF
UNKEY_BUILD_BACKEND=depot
DEPOT_TOKEN=$DEPOT_TOKEN
UNKEY_BUILD_S3_URL=$S3_URL
UNKEY_BUILD_S3_ACCESS_KEY_ID=$S3_ACCESS_KEY_ID
UNKEY_BUILD_S3_ACCESS_KEY_SECRET=$S3_ACCESS_KEY_SECRET
EOF
  echo "✓ Updated .env file with depot backend and S3 configuration"
else
  echo "Logging out from depot registry..."
  docker logout registry.depot.dev 2>/dev/null || true
  echo "✓ Switched to local Docker"

  # Write to .env file with default minio configuration
  cat >.env <<EOF
UNKEY_BUILD_BACKEND=docker
UNKEY_BUILD_S3_URL=http://s3:3902
UNKEY_BUILD_S3_ACCESS_KEY_ID=minio_root_user
UNKEY_BUILD_S3_ACCESS_KEY_SECRET=minio_root_password
EOF
  echo "✓ Updated .env file with docker backend and local minio configuration"
fi

echo "✓ Done! Build backend set to: $BACKEND"
