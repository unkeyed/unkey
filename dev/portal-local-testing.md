# Portal Local Testing Guide

## Prerequisites

### Portal app env vars

Copy the example and adjust if needed:

```bash
cp web/apps/portal/.env.example web/apps/portal/.env
```

Default values (work with `make up` / Tilt):

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_HOST` | `localhost:3900` | PlanetScale HTTP proxy |
| `DATABASE_USERNAME` | `unkey` | MySQL user |
| `DATABASE_PASSWORD` | `password` | MySQL password |
| `UNKEY_API_URL` | `http://localhost:7070` | Unkey API for session exchange |

### API server env var

The API server uses `UNKEY_PORTAL_BASE_URL` to construct session redirect
URLs in `createSession` responses. The TOML config references it via
`portal_base_url = "${UNKEY_PORTAL_BASE_URL}"`.

For Docker Compose, export it before starting the API:
```bash
export UNKEY_PORTAL_BASE_URL=http://localhost:3100
```

For Tilt / K8s, this is already set to `http://localhost:3100` in the API
ConfigMap (`dev/k8s/manifests/api.yaml`).

If unset, it defaults to `https://portal.unkey.com` (production).

## Seed Data

Seeding requires MySQL to be running. Start infra first, then seed.

For Docker Compose:
```bash
make up
go run . dev seed local --slug awesome --portal
```

For Tilt: run `make dev`, wait for MySQL to be healthy in the Tilt UI, then:
```bash
go run . dev seed local --slug awesome --portal
```

Save the root key printed at the end (e.g. `unkey_xxx`).

---

## Running Locally

### Option A: Tilt / K8s (full stack)

Runs the portal as a K8s deployment, same as production. Frontline handles
TLS and routing for `*.unkey.local`.

#### Setup (one-time)

```bash
dev/setup-wildcard-dns.sh   # *.unkey.local → 127.0.0.1 via dnsmasq
```

#### Start

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

The portal is port-forwarded to `http://localhost:3100`.

#### Create a session and open the portal

```bash
curl -s -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

Open `http://localhost:3100/?session=pst_xxx` in the browser.

---

### Option B: Docker Compose + pnpm dev (lightweight)

Faster iteration with hot reloading. No K8s required.

#### Start infrastructure + API

```bash
export UNKEY_PORTAL_BASE_URL=http://localhost:3100
docker compose -f dev/docker-compose.yaml up -d apiv2 apiv2_lb --wait
```

#### Start the portal

```bash
cd web && PORT=3100 pnpm --filter=@unkey/portal dev
```

Runs on `http://localhost:3100`. Port 3100 avoids conflict with the dashboard.

#### Create a session and open the portal

```bash
curl -s -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

Take the `sessionId` from the response and open:
```
http://localhost:3100/?session=pst_xxx
```

---

## Dogfooding on Unkey Deploy

Deploy the portal as a real app on Unkey Deploy to test the full production
stack (Frontline → Sentinel → portal container).

### 1. Create the project

In the Unkey dashboard, create a project for the portal (or use an existing
internal project). Add an app with:
- Dockerfile path: `web/apps/portal/Dockerfile`
- Docker context: `.` (repo root — the Dockerfile uses `web/` paths)
- Default branch: `main` (or your feature branch)

### 2. Configure environment variables

In the app's environment settings (production or preview), set:

| Variable | Value |
|----------|-------|
| `DATABASE_HOST` | Your staging/production PlanetScale host |
| `DATABASE_USERNAME` | DB username |
| `DATABASE_PASSWORD` | DB password |
| `UNKEY_API_URL` | `https://api.unkey.dev` (or staging API URL) |

### 3. Seed portal data

The target database needs `portal_configurations` and `portal_branding`
rows for the workspace you're testing with. If using the production DB,
insert them manually or via the dashboard (once the config UI is built).

For staging, you can run the seed tool against the staging DB:
```bash
go run . dev seed local \
  --slug awesome \
  --portal \
  --database-primary "<STAGING_DSN>"
```

### 4. Deploy

Push to the connected branch. The Deploy workflow builds the Docker image,
creates a deployment, and Frontline routes traffic to it. The deployment
gets a URL like `<app-slug>-<env>.unkey.com`.

### 5. Test the session flow

```bash
curl -s -X POST https://api.unkey.dev/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

The response URL will point to `portal.unkey.com` (or the custom domain if
configured). Open it in the browser — the full Frontline → portal flow runs
end-to-end.

### Notes

- The `portal_base_url` on the API server must match the deployment URL for
  session redirect URLs to work correctly.
- For preview environments, you may need to manually adjust the session URL
  domain if `portal_base_url` points to production.
- Custom domains work the same as any Deploy app — add the domain in the
  dashboard, set the CNAME, and certificates are provisioned automatically.

---

## Troubleshooting

- **Portal can't connect to DB**: Ensure `planetscale` container is running (`docker ps`)
- **Session creation fails with 401**: Re-run `go run . dev seed local --slug awesome --portal`
- **Session creation fails with 403**: Check `portal_configurations.enabled = TRUE`
- **"Invalid access" with session param**: Session may be expired (15 min TTL) or already exchanged (single-use). Create a fresh one.
- **Stale seed data**: `docker volume rm unkey_mysql` and restart
