# Portal Local Testing Guide

## Prerequisites

Start the local development environment:

```bash
# Start infrastructure
make up

# Start the API service (not included in make up)
docker compose -f dev/docker-compose.yaml up -d apiv2 apiv2_lb --wait
```

The MySQL seed script (`dev/05-seed-portal.sql`) must be run manually after
the Go seed tool. It creates portal-specific rows only:
- A `portal_configurations` row (`portal_awesome`) linked to the customer keyspace
- A `portal_branding` row with blue theme colors
- A `frontline_routes` row for `awesome.localhost` with `route_type='portal'`

The portal represents a customer offering self-service to their end users.

## Running the Portal

### Docker Compose

Not recommended for iterative development due to slow Docker builds.
If needed: `docker compose -f dev/docker-compose.yaml up -d portal --build`

### Standalone (recommended)

```bash
cd web && PORT=3100 pnpm --filter=@unkey/portal dev
```

Runs on `http://localhost:3100` with hot reloading. Port 3100 avoids conflict with the dashboard on 3000.

## Creating a Root Key

The Go seed tool creates the Awesome Corp workspace, keyspace, API, and root key:

```bash
go run . dev seed local --slug awesome
```

This prints a root key like `unkey_xxx_yyy`. Save it.

Then run the portal seed SQL to add portal config, branding, and frontline route:

```bash
docker exec -i mysql mysql -u root -proot < dev/05-seed-portal.sql
```

## End-to-End Flow

### 1. Create a session via the API

```bash
curl -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY_FROM_SEED>" \
  -d '{
    "externalId": "user_123",
    "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]
  }'
```

The response URL will use `https://` — change it to `http://` for local dev since there's no TLS locally.

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
