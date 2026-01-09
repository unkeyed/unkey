#!/bin/bash


pnpm install --frozen-lockfile

docker compose -f ../../../dev/docker-compose.yaml up -d planetscale agent clickhouse apiv2

# Write environment variables to .env if it doesn't exist
if [ ! -f .env ]; then
cat > .env << 'EOF'

DATABASE_HOST="localhost:3900"
DATABASE_USERNAME="unkey"
DATABASE_PASSWORD="password"

UNKEY_WORKSPACE_ID="ws_local_root"
UNKEY_API_ID="api_local_root_keys"

AUTH_PROVIDER="local"

AGENT_URL="http://localhost:8080"
AGENT_TOKEN="agent-auth-secret"

CLICKHOUSE_URL="http://default:password@localhost:8123"

CTRL_URL="http://localhost:7091"
CTRL_API_KEY="your-local-dev-key"

EOF
fi


pnpm run dev
