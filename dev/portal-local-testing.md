# Portal Local Testing Guide

## Seed Data

Both approaches require seeding the customer workspace and portal config.

```bash
# 1. Seed the Awesome Corp workspace, keyspace, API, and root key
go run . dev seed local --slug awesome

# 2. Seed portal config, branding, and frontline route
docker exec -i mysql mysql -u root -proot < dev/05-seed-portal.sql
```

Save the root key printed by step 1 (e.g. `unkey_xxx`).

---

## Option A: Tilt / K8s (full Frontline routing)

Portal URLs work exactly like production — Frontline handles TLS, hostname
routing, and path prefix stripping.

### Setup (one-time)

```bash
dev/setup-wildcard-dns.sh   # *.unkey.local → 127.0.0.1 via dnsmasq
```

### Start

```bash
make dev
```

In the Tilt UI (`http://localhost:10350`):
1. Trigger `build-portal-image`
2. Trigger `portal`

In a separate terminal:
```bash
sudo minikube tunnel
```

### Create a session and open the portal

```bash
curl -s -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

Copy the `url` from the response and open it directly in the browser.
It will be `https://awesome.unkey.local/portal?session=pst_xxx` — no edits needed.

---

## Option B: Docker Compose + pnpm dev (lightweight)

Faster iteration, no K8s required. Portal runs locally with hot reloading.
URLs need manual `https` → `http` adjustment.

### Start infrastructure + API

```bash
make up
docker compose -f dev/docker-compose.yaml up -d apiv2 apiv2_lb --wait
```

### Start the portal

```bash
cd web && PORT=3100 pnpm --filter=@unkey/portal dev
```

Runs on `http://localhost:3100`. Port 3100 avoids conflict with the dashboard.

### Create a session and open the portal

```bash
curl -s -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

The response URL will be `https://awesome.unkey.local/portal?session=pst_xxx`.
Replace with `http://localhost:3100/portal?session=pst_xxx` and open in browser.

---

## Troubleshooting

- **Portal can't connect to DB**: Ensure `planetscale` container is running (`docker ps`)
- **Session creation fails with 401**: Re-run `go run . dev seed local --slug awesome`
- **Session creation fails with 403**: Check `portal_configurations.enabled = TRUE`
- **"Portal not found"**: Re-run `docker exec -i mysql mysql -u root -proot < dev/05-seed-portal.sql`
- **"Invalid access" with session param**: Session may be expired (15 min TTL) or already exchanged (single-use). Create a fresh one.
- **Stale seed data**: `docker volume rm unkey_mysql` and restart
