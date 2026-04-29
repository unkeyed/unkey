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

For Tilt:
```bash
make dev
# wait for MySQL to be healthy in the Tilt UI
go run . dev seed local --slug awesome --portal
```

Save the root key printed at the end (e.g. `unkey_xxx`).

The portal config ID is `portal_<slug>` (e.g. `portal_awesome`). You'll
need it for `createSession` calls — it's also written to `dev/.env.seed`
as `UNKEY_PORTAL_CONFIG_ID`.

---

## Running Locally

### Option A: Tilt / K8s (full stack)

Runs the portal as a K8s deployment with a port-forward.

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
  -d '{"portalId": "portal_awesome", "externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

Open `http://localhost:3100/?session=pst_xxx` in the browser.

---

### Option B: Docker Compose + pnpm dev (lightweight)

Faster iteration with hot reloading via Vite. No K8s required.

#### Start infrastructure + API

```bash
export UNKEY_PORTAL_BASE_URL=http://localhost:3100
docker compose -f dev/docker-compose.yaml up -d apiv2 apiv2_lb --wait
```

#### Start the portal

```bash
pnpm --dir=web --filter=@unkey/portal dev
```

Runs on `http://localhost:3100`. Port 3100 avoids conflict with the dashboard.

#### Create a session and open the portal

```bash
curl -s -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"portalId": "portal_awesome", "externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

Take the `sessionId` from the response and open:
```text
http://localhost:3100/?session=pst_xxx
```

---

## Dogfooding on Unkey Deploy

Deploy the portal as an app on Unkey Deploy.

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
creates a deployment, and routes traffic to it. The deployment gets a URL
like `<app-slug>-<env>.unkey.com`.

### 5. Test the session flow

```bash
curl -s -X POST https://api.unkey.dev/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"portalId": "<PORTAL_CONFIG_ID>", "externalId": "user_123", "permissions": ["keys:read", "keys:create", "analytics:read", "docs:read"]}'
```

The response URL will point to the portal's deployment URL (or custom
domain if configured). Open it in the browser.

### Notes

- The `portal_base_url` on the API server must match the portal's
  deployment URL for session redirect URLs to work correctly.
- For preview environments, you may need to manually adjust the session URL
  domain if `portal_base_url` points to production.
- Custom domains work the same as any Deploy app — add the domain in the
  dashboard, set the CNAME, and certificates are provisioned automatically.

---

## Troubleshooting

- **Session creation fails with 400**: Make sure `portalId` is included in the request body
- **Session creation fails with 401**: Re-run `go run . dev seed local --slug awesome --portal`
- **Session creation fails with 403**: Check `portal_configurations.enabled = TRUE`
- **Session creation fails with 404**: The `portalId` doesn't exist or belongs to a different workspace. Verify the portal config was seeded and the root key matches the same workspace.
- **"Invalid access" with session param**: Session may be expired (15 min TTL) or already exchanged (single-use). Create a fresh one.
- **Stale seed data**: `docker volume rm unkey_mysql` and restart

---

## What the Seed Script Creates

`go run . dev seed local --slug <slug> --portal` sets up a complete local
environment in a single transaction. The `--slug` flag (default: `local`)
drives all generated IDs.

### Workspaces

| Workspace | ID | Purpose |
|-----------|----|---------|
| User workspace | `ws_<slug>` | Your working workspace |
| Root workspace | `ws_unkey` | Unkey internal workspace that owns root keys |

### Deploy resources (user workspace)

- **Project** (`<slug>-api`) with a **default app**
- **Environments**: `preview` and `production`
- Runtime settings, build settings, and regional settings for both environments
- A `local` region (upserted to avoid conflicts with Krane)

### Auth resources

- **Keyspaces**: `ks_<slug>_root_keys` (root, in `ws_unkey`) and `ks_<slug>` (user, in `ws_<slug>`)
- **APIs**: `api_unkey` (root) and `api_<slug>` (user), each linked to their keyspace
- **Root key** with all permissions (api, identity, RBAC, ratelimit, workspace, deploy)

### Portal resources (only with `--portal`)

- **Portal config** (`portal_<slug>`) linked to:
  - The user workspace (`ws_<slug>`)
  - The app created above (enables custom domain resolution)
  - The user keyspace (`ks_<slug>`) for legacy compatibility
- **Portal branding** (Unkey logo, blue color scheme)

### Output (`dev/.env.seed`)

```bash
UNKEY_WORKSPACE_ID=ws_<slug>
UNKEY_PROJECT_ID=<generated>
UNKEY_API_ID=api_<slug>
UNKEY_KEYSPACE_ID=ks_<slug>
UNKEY_ROOT_KEY=unkey_<generated>
UNKEY_PORTAL_CONFIG_ID=portal_<slug>   # only with --portal
```

You can run the seed multiple times with different slugs to create separate
workspaces. Upserts handle duplicates gracefully.
