# Portal Local Testing Guide

## Prerequisites

Start the local development environment:

```bash
# Docker Compose
make up

# OR Kubernetes with Tilt
make dev
```

The MySQL seed script (`dev/05-seed-portal.sql`) automatically creates a sample
customer workspace ("Awesome Corp" / `ws_awesome`) with:
- A workspace, keyspace (`ks_awesome_keys`), and API (`api_awesome`)
- A root key for the customer workspace (plaintext: `awesome_root_key_secret`)
- A `portal_configurations` row (`portal_awesome`) linked to the customer keyspace
- A `portal_branding` row with blue theme colors
- A `frontline_routes` row for `awesome.localhost` with `route_type='portal'`

The portal represents a customer offering self-service to their end users.

## Running the Portal

### Docker Compose

The portal runs automatically at `http://localhost:3100` when using `make up`.

### Tilt

In the Tilt UI (`http://localhost:10350`), manually trigger the `portal` resource.
It runs at `http://localhost:3100` with hot reloading.

### Standalone

```bash
cd web && pnpm --filter=@unkey/portal dev
```

## End-to-End Flow

### 1. Create a session via the API

Use the Awesome Corp root key to create a portal session for an end user:

```bash
curl -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer awesome_root_key_secret" \
  -d '{
    "externalId": "user_123",
    "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]
  }'
```

Response:
```json
{
  "sessionId": "sess_xxx",
  "url": "http://awesome.localhost:3100/portal?session=sess_xxx",
  "expiresAt": 1234567890000
}
```

### 2. Open the portal URL

Open the `url` from the response in your browser. The portal will:
1. Read the `?session=sess_xxx` query parameter
2. Call `POST /v2/portal.exchangeSession` to exchange it for a 24-hour browser session
3. Set an httpOnly cookie
4. Redirect to the first visible tab (based on permissions)

### 3. Verify the placeholder UI

You should see:
- A branded header with the logo and navigation tabs
- Tabs matching the permissions: Keys, Analytics, Docs
- Placeholder content on each page

## Manual Seed (if needed)

If the seed didn't run automatically (e.g., MySQL volume already existed), run:

```bash
docker exec -i mysql mysql -u root -proot < dev/05-seed-portal.sql
```

## Troubleshooting

- **Portal can't connect to DB**: Ensure `planetscale` container is running (`docker ps`)
- **Session creation fails with 401**: Verify the root key hash matches — re-run the seed SQL
- **Session creation fails with 403**: Check that `portal_configurations.enabled = TRUE`
- **"Invalid access" on portal**: Make sure you're using a valid, unexpired session URL
- **Stale seed data**: Remove the MySQL volume (`docker volume rm unkey_mysql`) and restart
