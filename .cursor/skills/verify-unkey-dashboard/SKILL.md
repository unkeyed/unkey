---
name: verify-unkey-dashboard
description: Launch and drive the Unkey Next.js dashboard at web/apps/dashboard (local auth, port 3000) the way a user would, capture proof, and clean up. Use when verifying dashboard UI, routes, keyspaces, keys, ratelimits, or the workspace switcher in github.com/unkeyed/unkey. Do not use it for WorkOS, production, or a second copy of the same stack.
---

# Verify the Unkey dashboard

Primary surface is the Next.js app `@unkey/dashboard` in `web/apps/dashboard`. Local mode (`AUTH_PROVIDER=local`) uses the built-in account `admin@example.com` (`user_local_admin`, org `org_localdefault`) and the `unkey-session` cookie. It does not load WorkOS AuthKit.

Secondary surfaces, not driven by this skill:

- `@unkey/design` (`web/apps/design`) dev server on port 4321.
- `@unkey/portal` (`web/apps/portal`) Vite dev server on port 3100. Customer portal, separate session.
- `web/apps/planetfall` is a stub (`next-env.d.ts` only) and has no package script.
- Go API from the dashboard compose file listens on port 7070. CLI is `mise run unkey`.

There is no Playwright or Cypress project. Dashboard tests are Vitest in jsdom (`web/apps/dashboard/vitest.config.ts`). UI driving uses the browser's accessibility tree (role and accessible name). Vitest is non-UI proof only.

## Launch

One instance only. `web/apps/dashboard/dev/docker-compose.yaml` sets container names `mysql`, `clickhouse`, `redis`, `vault`, `restate`, `ctrl-api`, `ctrl-worker`, and `api`, and publishes ports 3306, 8123, 9000, 6379, 8060, 8081, 9070, 7091, 9080, and 7070. `next dev` listens on port 3000. A second `mise run dashboard` cannot run beside the first. If doctor reports `refuse-shared`, stop. Do not attach to a server you did not start.

Prepare `web/apps/dashboard/.env` from the repo examples. Do not invent secrets and do not copy a hosted URL.

```bash
test -f web/apps/dashboard/.env || cp web/apps/dashboard/.env.example web/apps/dashboard/.env
```

Set these before a canonical launch. Values below are the local compose and example files, not production:

- `AUTH_PROVIDER=local`
- `UNKEY_API_URL=http://localhost:7070` (if this is unset, `lib/env.ts` defaults to `https://api.unkey.com`; doctor refuses that)
- `VAULT_URL=http://localhost:8060` and `VAULT_TOKEN` from `web/apps/dashboard/.env.example`
- `DATABASE_PRIMARY` from `web/apps/dashboard/dev/.env.example` (`mysql://unkey:password@localhost:3306/unkey`)
- `CTRL_URL=http://127.0.0.1:7091` and `CTRL_API_KEY` from the example, only if you set them
- Leave `FLAGS_LOCAL_OVERRIDES` empty unless a feature file names a flag. `projects-nav` and `portal-management` default off.

`UNKEY_WORKSPACE_ID` and `UNKEY_API_ID` may stay empty for the sign-in document. They are required keys in the schema but empty strings parse. After a successful seed, `dev/.env.seed` contains the generated ids (`ws_local`, `api_local` for the default slug). Copy `UNKEY_WORKSPACE_ID`, `UNKEY_API_ID`, and `UNKEY_ROOT_KEY` from that file into `web/apps/dashboard/.env` before exercising server-side Unkey client calls. Do not print `UNKEY_ROOT_KEY`.

Canonical start, from the repo root, after doctor says `safe-to-launch`:

```bash
.cursor/skills/verify-unkey-dashboard/scripts/launch.sh
```

That script runs `mise run dashboard` (Go build, web install, `docker compose -f web/apps/dashboard/dev/docker-compose.yaml up -d --wait`, `bin/unkey dev seed local`, then `pnpm --dir=web turbo run dev --filter=@unkey/dashboard`). It records the session-leader pid in `.cursor/skills/verify-unkey-dashboard/.run/dashboard.pid`.

Readiness: poll until this exits 0 and prints `verdict: drivable`:

```bash
.cursor/skills/verify-unkey-dashboard/scripts/doctor.sh ready
```

The HTTP signal inside that check is `GET http://127.0.0.1:3000/auth/sign-in` returning a document that contains `Local dashboard` and `Continue to dashboard`.

Image builds and the first Next compile take long enough that a short curl loop is not a failure. Read `.cursor/skills/verify-unkey-dashboard/.run/dashboard.log`. If the pid exits, run cleanup and treat the log tail as the blocker.

This VM class often has no Docker daemon. When `doctor.sh preflight` exits 4 because `docker` is missing or `docker info` fails, do not start the canonical launch. You may prove only the auth landing:

```bash
.cursor/skills/verify-unkey-dashboard/scripts/launch-auth-landing.sh
```

That runs `mise exec -- pnpm --dir=web/apps/dashboard dev` and waits until doctor prints `verdict: auth-landing-only`. It does not start MySQL or the API. Drive `features/auth-gate.md` only. Do not activate `Continue to dashboard`.

Teardown is `scripts/cleanup.sh` in the Cleanup section.

## Doctor

Read-only. It prints a `verdict:` line and never starts or stops processes.

```bash
.cursor/skills/verify-unkey-dashboard/scripts/doctor.sh preflight
.cursor/skills/verify-unkey-dashboard/scripts/doctor.sh preflight-auth
.cursor/skills/verify-unkey-dashboard/scripts/doctor.sh ready
```

| Command | Exit 0 | Other exits |
| --- | --- | --- |
| `preflight` | `safe-to-launch` | 2 `refuse-shared`, 4 `blocked` (no mise, no docker, non-local URL, `AUTH_PROVIDER` not `local`, empty `DATABASE_PRIMARY`) |
| `preflight-auth` | `safe-to-launch-auth` | 2 shared listener, 4 blocked env |
| `ready` | `drivable` or `auth-landing-only` | 2 unknown listener on port 3000, 3 nothing running, 4 sign-in document is not the local landing |

`drivable` means the pid this skill recorded is alive, port 3000 serves the local landing, and ports 3306 and 7070 are open. `auth-landing-only` means the landing is up and MySQL or the API is not. Authenticated routes are not worth driving in that state: `app/(app)/layout.tsx` loads the workspace over tRPC, and `lib/db.ts` parses `DATABASE_PRIMARY` at import.

`refuse-shared` is terminal for this run. Do not kill an unknown pid. Do not run a second compose project.

Doctor prints hosts and `AUTH_PROVIDER`. It does not print database URLs, tokens, or root keys.

These lines in `.run/dashboard.log` are normal on a local boot and are not a failed drive:

- `[flags] failed to create vercel adapter, falling back to noop` when `FLAGS` is unset (`lib/flags/plumbing.ts`).
- `You are rendering the Vercel Toolbar in development, but the configuration is missing.` The toolbar plugin is in `next.config.js`; the project is not linked with `vercel link`.
- `Blocked cross-origin request to Next.js dev resource /_next/hmr from "127.0.0.1"`. Doctor and the browser use `http://127.0.0.1:3000`. The document still renders. A blank page is a failure; that log line alone is not.

## Drive

Use a browser attached to `http://127.0.0.1:3000`. Prefer `getByRole` names from the feature files. Do not click by coordinates. Do not drive `https://app.unkey.com` or a WorkOS hosted screen.

With `projects-nav` off (the default), the sidebar from `buildWorkspaceSections` in `web/apps/dashboard/lib/navigation/leaves.ts` is:

| Accessible name | Href after seed slug `local` |
| --- | --- |
| Projects | `/local/projects` |
| Keyspaces (APIs) | `/local/apis` |
| Ratelimit | `/local/ratelimits` |
| Authorization | `/local/authorization/roles` |
| Logs | `/local/logs` |
| Identities | `/local/identities` |
| Audit Log | `/local/audit` |
| Settings | `/local/settings/general` |

Seed (`mise run unkey -- dev seed local`, already invoked by `mise run dashboard`) creates workspace slug `local`, display name `Org Local`, id `ws_local`, API `api_local` named `Local API`, keyspace `ks_local`. Keys live at `/local/apis/api_local/keys/ks_local`.

The sign-in link `Continue to dashboard` points at `/apis`. That path is `[workspaceSlug]` with slug `apis`, and `app/(app)/[workspaceSlug]/page.tsx` then replaces to `/{session workspace slug}/projects`. It is not the keyspace list. The keyspace list is `/local/apis` or the sidebar link `Keyspaces (APIs)`.

Workspace-independent docs links use `/~/...`, which `app/(app)/~/[[...path]]/page.tsx` redirects through the session workspace. `/~/apis` becomes `/local/apis` after seed.

When doctor says `auth-landing-only`, the only drive is `features/auth-gate.md`. When doctor says `drivable`, follow one feature file end to end.

Non-UI proof, which does not render the dashboard, for the auth session contract:

```bash
mise exec -- pnpm --dir=web/apps/dashboard exec vitest run lib/auth/tests/local.test.ts lib/navigation/routes/auth.test.ts
```

## Evidence

Write a new directory `.cursor/skills/verify-unkey-dashboard/evidence/<UTC timestamp>/`. Do not overwrite an older run.

Capture:

- `doctor-preflight.txt` and `doctor-ready.txt` (stdout and stderr)
- the launch log tail (`dashboard.log` copied here; the live log stays in `.run/`)
- the URL you opened
- the control you used (role and accessible name) and the resulting state (heading, toast, or URL after navigation)
- a screenshot or the saved HTML of that state
- for a mutation, the side effect (new row, toast text, or the Vitest pass output when the UI could not run)
- `cleanup.txt` and a second `doctor.sh ready` after cleanup showing `not-running`

A passing proof shows a real user path: an action and the state it produced. Mock only at a boundary the app already has (local auth is that boundary; do not stub tRPC for a UI proof). A Vitest run proves the module you named, not the screen.

## Cleanup

```bash
.cursor/skills/verify-unkey-dashboard/scripts/cleanup.sh
```

The script signals the recorded session leader and, when mode is `canonical` and docker answers, runs `docker compose -f web/apps/dashboard/dev/docker-compose.yaml down`. It does not kill by process name. It does not remove `evidence/`. After it returns, `doctor.sh ready` must exit 3 (`not-running`). If port 3000 is still open, it belongs to someone else: stop and report `refuse-shared`. Do not hunt for `next` or `node` by name.

## Helpers

Invoked from this skill:

- `.cursor/skills/verify-unkey-dashboard/scripts/doctor.sh`
- `.cursor/skills/verify-unkey-dashboard/scripts/launch.sh`
- `.cursor/skills/verify-unkey-dashboard/scripts/launch-auth-landing.sh`
- `.cursor/skills/verify-unkey-dashboard/scripts/cleanup.sh`

`scripts/lib.sh` is sourced by those four. It is not an entrypoint. `.run/` is gitignored runtime state (pid, mode, log).
