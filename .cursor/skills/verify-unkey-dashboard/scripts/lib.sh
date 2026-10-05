#!/usr/bin/env bash
# Shared paths for the verify-unkey-dashboard scripts. Sourced, not executed.
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT="$(cd "$SKILL_DIR/../../.." && pwd)"
RUN_DIR="$SKILL_DIR/.run"
ENV_FILE="$ROOT/web/apps/dashboard/.env"
COMPOSE_FILE="$ROOT/web/apps/dashboard/dev/docker-compose.yaml"
PID_FILE="$RUN_DIR/dashboard.pid"
MODE_FILE="$RUN_DIR/mode"
LOG_FILE="$RUN_DIR/dashboard.log"
PORT=3000
BASE_URL="http://127.0.0.1:${PORT}"

export PATH="${HOME}/.local/bin:${PATH}"

mkdir -p "$RUN_DIR"

die() {
  printf 'verdict: %s\n%s\n' "$1" "$2" >&2
  exit "$3"
}

env_value() {
  local key="$1" line raw
  [[ -f "$ENV_FILE" ]] || return 0
  line="$(grep -E "^${key}=" "$ENV_FILE" | tail -n 1 || true)"
  raw="${line#*=}"
  raw="${raw%$'\r'}"
  if [[ "$raw" == \"*\" && "$raw" == *\" ]]; then
    raw="${raw:1:${#raw}-2}"
  elif [[ "$raw" == \'*\' && "$raw" == *\' ]]; then
    raw="${raw:1:${#raw}-2}"
  fi
  printf '%s' "$raw"
}

db_host() {
  local url="$1" rest hostport
  [[ -n "$url" ]] || return 0
  rest="${url#*://}"
  rest="${rest#*@}"
  hostport="${rest%%/*}"
  hostport="${hostport%%\?*}"
  printf '%s' "${hostport%%:*}"
}

port_open() {
  local port="$1"
  (echo >/dev/tcp/127.0.0.1/"$port") >/dev/null 2>&1
}

pid_alive() {
  local pid="$1"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

recorded_pid() {
  [[ -f "$PID_FILE" ]] || return 0
  tr -d '[:space:]' <"$PID_FILE"
}

recorded_mode() {
  [[ -f "$MODE_FILE" ]] || return 0
  tr -d '[:space:]' <"$MODE_FILE"
}

# Fixed by web/apps/dashboard/dev/docker-compose.yaml. A second dashboard
# cannot bind these, and the container_name values collide globally.
SHARED_PORTS=(3000 3306 6379 7070 7091 8060 8081 8123 9000 9070 9080)
SHARED_CONTAINERS=(mysql clickhouse redis vault restate ctrl-api ctrl-worker api)

foreign_ports() {
  local port
  for port in "${SHARED_PORTS[@]}"; do
    if port_open "$port"; then
      printf '%s\n' "$port"
    fi
  done
}

foreign_containers() {
  command -v docker >/dev/null 2>&1 || return 0
  docker info >/dev/null 2>&1 || return 0
  local name
  local running
  running="$(docker ps -a --format '{{.Names}}' 2>/dev/null || true)"
  for name in "${SHARED_CONTAINERS[@]}"; do
    if grep -Fxq "$name" <<<"$running"; then
      printf '%s\n' "$name"
    fi
  done
}

require_local_auth() {
  local provider
  [[ -f "$ENV_FILE" ]] || die blocked "Missing $ENV_FILE. Copy web/apps/dashboard/.env.example to web/apps/dashboard/.env. Do not point it at a shared or production database." 4
  provider="$(env_value AUTH_PROVIDER)"
  if [[ "$provider" != "local" ]]; then
    die blocked "AUTH_PROVIDER is '${provider:-unset}'. This skill drives only AUTH_PROVIDER=local. It does not sign in through WorkOS." 4
  fi
  if [[ -z "$(env_value VAULT_URL)" || -z "$(env_value VAULT_TOKEN)" ]]; then
    die blocked "VAULT_URL and VAULT_TOKEN must be set. Use the local values in web/apps/dashboard/.env.example. The root layout parses them on every request, including /auth/sign-in." 4
  fi
  local db host
  db="$(env_value DATABASE_PRIMARY)"
  if [[ -n "$db" ]]; then
    host="$(db_host "$db")"
    case "$host" in
      localhost | 127.0.0.1) ;;
      *) die blocked "DATABASE_PRIMARY host is '${host:-unknown}', not localhost. Refusing to drive a non-local database." 4 ;;
    esac
  fi
  local ch ch_host
  ch="$(env_value CLICKHOUSE_URL)"
  if [[ -n "$ch" ]]; then
    ch_host="$(db_host "$ch")"
    case "$ch_host" in
      localhost | 127.0.0.1) ;;
      *) die blocked "CLICKHOUSE_URL host is '${ch_host:-unknown}', not localhost. Refusing to drive a non-local ClickHouse." 4 ;;
    esac
  fi
  local api api_host
  api="$(env_value UNKEY_API_URL)"
  if [[ -z "$api" ]]; then
    die blocked "UNKEY_API_URL is unset. lib/env.ts defaults it to https://api.unkey.com. Set http://localhost:7070 from web/apps/dashboard/.env.example." 4
  fi
  api_host="$(db_host "$api")"
  case "$api_host" in
    localhost | 127.0.0.1) ;;
    *) die blocked "UNKEY_API_URL host is '${api_host:-unknown}', not localhost. Refusing to call a non-local API." 4 ;;
  esac
  local ctrl ctrl_host
  ctrl="$(env_value CTRL_URL)"
  if [[ -n "$ctrl" ]]; then
    ctrl_host="$(db_host "$ctrl")"
    case "$ctrl_host" in
      localhost | 127.0.0.1) ;;
      *) die blocked "CTRL_URL host is '${ctrl_host:-unknown}', not localhost. Refusing to call a non-local control plane." 4 ;;
    esac
  fi
}

our_instance_owns_listeners() {
  local pid mode
  pid="$(recorded_pid)"
  mode="$(recorded_mode)"
  pid_alive "$pid" || return 1
  [[ "$mode" == "canonical" || "$mode" == "auth-landing" ]] || return 1
  port_open "$PORT"
}
