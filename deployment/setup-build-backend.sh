#!/bin/bash
set -e

readonly CONFIG_FILE="./config/depot.json"

print_usage() {
  echo "Usage: $0 [depot|docker]"
  echo ""
  echo "Examples:"
  echo "  $0 depot   # Switch to Depot builds"
  echo "  $0 docker  # Switch to local Docker builds"
  exit 1
}

load_from_config() {
  local key="$1"
  grep -o "\"$key\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "$CONFIG_FILE" | sed "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/"
}

read_config_file() {
  [ ! -f "$CONFIG_FILE" ] && return

  echo "Reading configuration from $CONFIG_FILE..."

  DEPOT_TOKEN=$(load_from_config "token")
  S3_URL=$(load_from_config "s3_url")
  S3_ACCESS_KEY_ID=$(load_from_config "s3_access_key_id")
  S3_ACCESS_KEY_SECRET=$(load_from_config "s3_access_key_secret")

  [ -n "$DEPOT_TOKEN" ] && echo "✓ Token loaded from config file"
  [ -n "$S3_URL" ] && echo "✓ S3 URL loaded from config file"
  [ -n "$S3_ACCESS_KEY_ID" ] && echo "✓ S3 Access Key ID loaded from config file"
  [ -n "$S3_ACCESS_KEY_SECRET" ] && echo "✓ S3 Access Key Secret loaded from config file"
}

prompt_if_empty() {
  local var_name="$1"
  local prompt_text="$2"
  local is_secret="${3:-false}"

  local current_value
  eval "current_value=\$$var_name"

  if [ -z "$current_value" ]; then
    echo "$prompt_text"
    if [ "$is_secret" = "true" ]; then
      read -r -s new_value
      echo ""
    else
      read -r new_value
    fi

    if [ -z "$new_value" ]; then
      echo "Error: $var_name is required"
      exit 1
    fi

    eval "$var_name=\"$new_value\""
  fi
}

validate_depot_token() {
  if [[ ! "$DEPOT_TOKEN" =~ ^depot_org_ ]]; then
    echo "Error: Invalid Depot token format. Token must start with 'depot_org_'"
    exit 1
  fi
}

login_depot_registry() {
  if echo "$DEPOT_TOKEN" | docker login registry.depot.dev -u x-token --password-stdin >/dev/null 2>&1; then
    echo "✓ Logged into depot registry"
  else
    echo ""
    echo "Error: Docker login failed. Invalid token?"
    exit 1
  fi
}

setup_depot() {
  read_config_file

  # Fall back to environment variables
  DEPOT_TOKEN="${DEPOT_TOKEN:-${DEPOT_TOKEN:-}}"
  S3_URL="${S3_URL:-${UNKEY_BUILD_S3_URL:-}}"
  S3_ACCESS_KEY_ID="${S3_ACCESS_KEY_ID:-${UNKEY_BUILD_S3_ACCESS_KEY_ID:-}}"
  S3_ACCESS_KEY_SECRET="${S3_ACCESS_KEY_SECRET:-${UNKEY_BUILD_S3_ACCESS_KEY_SECRET:-}}"

  prompt_if_empty "DEPOT_TOKEN" "Enter your Depot token:" true
  validate_depot_token
  login_depot_registry

  echo ""
  echo "S3 Configuration for Depot:"
  echo "----------------------------"

  prompt_if_empty "S3_URL" "Enter S3 URL (e.g., https://aaar2.cloudflarestorage.com):"
  prompt_if_empty "S3_ACCESS_KEY_ID" "Enter S3 Access Key ID:"
  prompt_if_empty "S3_ACCESS_KEY_SECRET" "Enter S3 Access Key Secret:" true

  cat >.env <<EOF
UNKEY_BUILD_BACKEND=depot
DEPOT_TOKEN=$DEPOT_TOKEN
UNKEY_BUILD_S3_EXTERNAL_URL=
UNKEY_BUILD_S3_URL=$S3_URL
UNKEY_BUILD_S3_ACCESS_KEY_ID=$S3_ACCESS_KEY_ID
UNKEY_BUILD_S3_ACCESS_KEY_SECRET=$S3_ACCESS_KEY_SECRET

# Krane registry configuration - allows Krane to pull images built by Depot
UNKEY_REGISTRY_URL=registry.depot.dev
UNKEY_REGISTRY_USERNAME=x-token
UNKEY_REGISTRY_PASSWORD=$DEPOT_TOKEN
EOF

  echo "✓ Updated .env file with depot backend and S3 configuration"
  echo "✓ Configured Krane to pull images from Depot registry"
}

setup_docker() {
  echo "Logging out from depot registry..."
  docker logout registry.depot.dev 2>/dev/null || true
  echo "✓ Switched to local Docker"

  cat >.env <<EOF
UNKEY_BUILD_BACKEND=docker
UNKEY_BUILD_S3_EXTERNAL_URL=http://localhost:3902
UNKEY_BUILD_S3_URL=http://s3:3902
UNKEY_BUILD_S3_ACCESS_KEY_ID=minio_root_user
UNKEY_BUILD_S3_ACCESS_KEY_SECRET=minio_root_password

# Krane registry configuration - not needed for local Docker builds
# UNKEY_REGISTRY_URL=
# UNKEY_REGISTRY_USERNAME=
# UNKEY_REGISTRY_PASSWORD=
EOF

  echo "✓ Updated .env file with docker backend and local minio configuration"
}

main() {
  local backend="${1:-}"

  if [ "$backend" != "depot" ] && [ "$backend" != "docker" ]; then
    print_usage
  fi

  if [ "$backend" = "depot" ]; then
    setup_depot
  else
    setup_docker
  fi

  echo "✓ Done! Build backend set to: $backend"
}

main "$@"
