# Portal App Development Guide

## Product Overview

The Customer Portal (`web/apps/portal`) is a white-labeled web app that Unkey customers embed for their end users. It provides self-service API key management, usage analytics, and API documentation — without the customer building any UI.

Authentication uses a Stripe-style session flow:
1. Customer backend calls `POST /v2/portal.createSession` with a root key, slug, externalId, and RBAC permissions
2. API returns a short-lived session token (15 min, single-use) and a portal URL
3. Customer redirects end user to the portal URL
4. Portal exchanges the token for a 24-hour browser session (httpOnly cookie)
5. Portal renders tabs based on the session's RBAC permissions

The portal has **not shipped yet** — all tab pages are placeholder UI.

## Technology Stack

| Layer | Technology |
|-------|-----------|
| Framework | TanStack Start (React + Vite SSR) |
| Router | TanStack Router (file-based routing) |
| UI | React 19, Tailwind CSS 4, `@unkey/ui`, `@unkey/icons` |
| Database | Drizzle ORM + mysql2 (read-only, for session/config lookups) |
| API Client | `@unkey/api` SDK (for key management, analytics) |
| Validation | Zod 4 |
| Testing | Vitest 3 |
| Build | Vite 6, Docker multi-stage |
| Port | 3100 (avoids conflict with dashboard on 3000) |

## Directory Structure

```
web/apps/portal/
├── src/
│   ├── components/          # Shared UI components
│   │   ├── portal-header.tsx    # Branded nav with permission-derived tabs
│   │   └── preview-banner.tsx   # "Preview mode" indicator
│   ├── lib/                 # Server-side utilities
│   │   ├── db.ts                # Drizzle connection pool
│   │   ├── env.ts               # Zod-validated env vars
│   │   ├── permissions.ts       # Tab derivation from RBAC tuples
│   │   ├── permissions.test.ts  # Unit tests for tab derivation
│   │   ├── portal-config.ts     # Load portal config + branding from DB
│   │   ├── session.ts           # Session exchange + cookie management
│   │   └── unkey.ts             # @unkey/api client factory
│   ├── routes/
│   │   ├── __root.tsx           # HTML shell, fonts, meta
│   │   ├── index.tsx            # Entry: exchanges session token → redirect
│   │   ├── _portal.tsx          # Authenticated layout (session guard)
│   │   └── _portal/
│   │       ├── keys.tsx         # API Keys tab (placeholder)
│   │       ├── analytics.tsx    # Analytics tab (placeholder)
│   │       └── docs.tsx         # Documentation tab (placeholder)
│   └── styles/
│       └── tailwind.css
├── .env.example
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## Key Concepts

### Session Flow

The `index.tsx` route handles the initial session exchange:
- Reads `?session=pst_xxx` from the URL
- Calls `exchangeSession` (server function) which POSTs to the API and sets an httpOnly cookie
- On success, resolves permissions → picks the default tab → navigates

The `_portal.tsx` layout guard (`beforeLoad`) checks for a valid session cookie on every navigation. If missing/expired, redirects to `/`.

### RBAC Permissions & Tab Visibility

Permissions are RBAC tuples in the format `{resourceType}.{resourceId}.{action}`. Tab visibility is derived from the **action segment** (third dot-separated part):

| Action | Tab |
|--------|-----|
| `read_key`, `create_key`, `update_key`, `delete_key` | Keys |
| `read_analytics` | Analytics |
| Any permission present | Docs |

The `deriveVisibleTabs` function in `src/lib/permissions.ts` implements this logic. Permissions with fewer than 3 segments are silently ignored (defensive fallback).

### Branding

Portal configs can have custom branding (logo URL, primary color). The `_portal.tsx` layout injects CSS custom properties (`--portal-primary`, `--portal-secondary`) that components reference for theming.

### Portal Configuration

Each portal config is identified by a **slug** (3–64 chars, lowercase alphanumeric + hyphens, unique per workspace). The slug is what API consumers pass in `createSession` requests. The portal itself resolves config by `portal_config_id` from the session record.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `UNKEY_API_URL` | No | `https://api.unkey.dev` | API endpoint for session exchange |
| `DATABASE_HOST` | Yes | — | MySQL host:port for session/config reads |
| `DATABASE_USERNAME` | Yes | — | MySQL username |
| `DATABASE_PASSWORD` | Yes | — | MySQL password |

## Development

```bash
# Start infrastructure (MySQL, API server)
make up

# Seed portal data
go run . dev seed local --slug awesome --portal

# Start portal with hot reload
pnpm --dir=web --filter=@unkey/portal dev

# Create a test session
curl -s -X POST http://localhost:7070/v2/portal.createSession \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ROOT_KEY>" \
  -d '{"slug": "awesome", "externalId": "user_123", "permissions": ["api.*.read_key", "api.*.create_key", "api.*.read_analytics"]}'

# Open in browser
open http://localhost:3100/?session=pst_xxx
```

See `dev/portal-local-testing.md` for full setup instructions including Tilt/K8s and Docker Compose options.

## Conventions

### File Organization
- Server-only code in `src/lib/` — use `@tanstack/react-start/server-only` import guard
- Route components in `src/routes/` following TanStack Router file-based conventions
- Shared UI components in `src/components/`
- No `use client` directive needed — TanStack Start handles the client/server boundary

### Styling
- Use Tailwind utility classes with Radix Colors (same 12-step scale as dashboard)
- Branding colors via CSS custom properties (`var(--portal-primary, fallback)`)
- Use `color-mix()` for opacity variants of brand colors

### Testing
- Unit tests colocated with source (e.g., `permissions.test.ts` next to `permissions.ts`)
- Run with `pnpm --dir=web --filter=@unkey/portal test`
- Use Vitest 3 (not the dashboard's Vitest 1.6)

### Security
- Session tokens are never exposed to client-side JavaScript (httpOnly cookie)
- The portal reads the database directly for session validation (no API round-trip for auth)
- `noindex, nofollow` meta tag prevents search engine indexing
- All API calls from the portal use the session token as auth (via `createPortalClient`)

## API Endpoints (Backend)

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `POST /v2/portal.createSession` | Root key | Create session token + portal URL |
| `POST /v2/portal.exchangeSession` | None (token is proof) | Exchange token → browser session |

Both endpoints live in `svc/api/routes/v2_portal_*` on the Go side.

## Database Tables

| Table | Purpose |
|-------|---------|
| `portal_configurations` | Portal config per workspace (slug, enabled, branding ref) |
| `portal_branding` | Logo URL, primary color |
| `portal_session_tokens` | Short-lived tokens (15 min, single-use) |
| `portal_sessions` | Long-lived browser sessions (24 hr) |

The portal app has **read-only** database access. Writes happen through the API server.
