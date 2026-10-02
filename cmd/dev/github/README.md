# GitHub Dev Tools

Local development tools for setting up and testing GitHub App-triggered deployments.

## Commands

- `go run ./build/cli dev github setup`: create a GitHub App via manifest flow and write all credentials automatically
- `go run ./build/cli dev github tunnel`: start an ngrok tunnel and update the GitHub App webhook URL automatically
- `go run ./build/cli dev github trigger-webhook`: simulate a GitHub push webhook to trigger a deployment
- `mise run unkey -- dev github relay-events`: pull registered events from the installation relay into the local control API

---

## Setup

### Step 1: Create the GitHub App

```bash
go run ./build/cli dev github setup --app-name my-unkey-dev
```

This opens a browser, walks you through GitHub's App creation UI, then writes:

- `dev/.env.github`: app ID, webhook secret, app name
- `dev/.github-private-key.pem`: private key for ctrl-worker
- `web/apps/dashboard/.github-private-key.pem`: private key for the dashboard
- `web/apps/dashboard/.env`: `GITHUB_APP_ID`, `NEXT_PUBLIC_GITHUB_APP_NAME`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`

### Step 2: Start the dev environment

```bash
mise run dev
```

Without `GITHUB_INSTALL_RELAY_URL`, Tilt starts a `github-tunnel` resource that runs ngrok against ctrl-api and patches the GitHub App's webhook URL to point at the public ngrok address. Relay users must not run this tunnel against the shared App.

### Step 3: Seed the database

```bash
go run ./build/cli dev seed local
```

### Step 4: Trigger a deployment

```bash
go run ./build/cli dev github trigger-webhook \
  --project local-api \
  --repository owner/repo
```

Omitting `--commit-sha` deploys the HEAD of the repo's default branch (resolved via the GitHub API). Pass `--commit-sha <40-char-sha>` and `--branch <name>` to pin a specific commit.

---

## Install through a shared development relay

Use a separate development GitHub App and the standalone
[`unkeyed/github-relay`](https://github.com/unkeyed/github-relay) service for
dashboards with changing public origins. Do not change the production App.

Set both the development App's first **Callback URL** and **Setup URL** to
`https://<relay-host>/api/integrations/github/relay/callback`. Enable
**Request user authorization (OAuth) during installation** and
**Redirect on update**. Disable wildcard callback matching. The Setup URL
alone does not configure OAuth callbacks.

Set the dashboard's `DASHBOARD_BASE_URL` to its exact public HTTPS origin and
`GITHUB_INSTALL_RELAY_URL` to the relay origin, without trailing slashes.
Supply either a scoped `GITHUB_INSTALL_RELAY_TOKEN` or a
`GITHUB_INSTALL_RELAY_ADMIN_TOKEN`, never both. Admin mode automatically enrolls
the dashboard origin and is for trusted development only. The dashboard still
needs `GITHUB_APP_ID`, `UNKEY_GITHUB_PRIVATE_KEY_PEM`, and
`NEXT_PUBLIC_GITHUB_APP_NAME` for the same App. Keep OAuth client credentials
on the relay.

Without relay settings, direct installation uses the dashboard callback at
`/integrations/github/callback` and requires its existing OAuth client settings.
Without GitHub credentials, installation controls are unavailable; container
image deployment remains available.

## Receive events through a shared development relay

The standalone `unkeyed/github-relay` service receives GitHub webhooks at
`/webhooks/github`. Each opted-in development environment pulls its own queued
events and forwards them to its loopback control API. GitHub does not need access
to an Amp portal, and orbs do not overwrite the shared App's webhook URL.

1. Deploy the relay version with webhook support and its database migration.
   Set the relay's `GITHUB_WEBHOOK_SECRET` to the GitHub App webhook secret.
2. Set the development GitHub App's **Webhook URL** to
   `https://<relay-host>/webhooks/github`, use JSON content, and subscribe to
   push and pull request events. Changing this URL affects every user of the App.
3. Set `UNKEY_GITHUB_APP_WEBHOOK_SECRET` in `dev/.env.github` to the same secret
   and restart the local control API after updating its Kubernetes secret.
4. Set `GITHUB_INSTALL_RELAY_URL` and `DASHBOARD_BASE_URL` to the exact HTTPS
   relay and dashboard origins, without trailing slashes. Supply exactly one
   credential through the environment: `GITHUB_INSTALL_RELAY_TOKEN`, or
   `GITHUB_INSTALL_RELAY_ADMIN_TOKEN` for trusted development only. Never give
   the admin token to an untrusted preview.
5. Start the forwarder, then complete the GitHub installation or authorization
   flow from this dashboard. The relay records the repositories that the user
   can access within the installation. A database seed or a caller-provided ID does not
   authorize a subscription. Existing installations must complete this flow again
   after the relay upgrade if no repository access proof exists. Reconnect every
   24 hours to renew that proof, and after adding repositories. Enrollment renewal
   alone does not extend repository access.

```bash
mise run unkey -- dev github relay-events
```

The command reads credentials from the environment, not flags. It forwards to
`http://localhost:7091/webhooks/github` by default. Use `--webhook-url` for another
loopback port. HTTP redirects are not followed, and credentials are never sent
to the local receiver.

For a supervised Tilt process, set `GITHUB_INSTALL_RELAY_WEBHOOKS=true` before
starting `mise run dev` or `mise run dev-orb`. If `DASHBOARD_BASE_URL` is unset,
Tilt reads it from dashboard `.env.local`. Relay credentials and URL come from
Tilt's process environment. Either this opt-in or a relay URL in that environment
disables the ngrok tunnel.

Delivery preserves the raw payload, GitHub signature, event type, and delivery
ID. Only a 2xx response from the control API acknowledges the leased event.
Failures retry with backoff. A crash after acceptance but before acknowledgment
can deliver the event again; the control API uses GitHub's delivery ID for
Restate idempotency. Each environment has an independent queue. Do not attach
multiple environments to the same control-plane database.

The relay retains payloads for 24 hours, permits at most 20 claims per delivery,
and retains deduplication IDs for seven days. It rejects bodies above 2 MiB with
413. It forwards only `push` and `pull_request` events, and does not replay old
events to newly linked environments. Stop the forwarder to pause consumption;
revoke the environment registration to remove its subscriptions and queued data.

---

## Flags

### `setup`

| Flag | Description | Default |
|------|-------------|---------|
| `--app-name` | GitHub App name (must be globally unique) | `unkey-dev` |
| `--webhook-url` | Initial webhook URL; Tilt's `github-tunnel` overwrites this on boot | `https://example.com/webhooks/github` |
| `--port` | Local callback server port | `9999` |
| `--out-dir` | Where to write `dev/` credentials | `dev` |

### `tunnel`

Tilt's `github-tunnel` resource invokes this automatically when `dev/.env.github` and `dev/.github-private-key.pem` are present. The command is kept for manual use (debugging, running outside Tilt).

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Local port to tunnel | `7091` |
| `--env-file` | Path to `.env.github` | `dev/.env.github` |
| `--pem-file` | Path to `.github-private-key.pem` | `dev/.github-private-key.pem` |

Requires `ngrok` to be installed.

### `trigger-webhook`

| Flag | Description | Default |
|------|-------------|---------|
| `--project` | Unkey project slug (e.g. `local-api`) | **required** |
| `--repository` | Full repository name (`owner/repo`) | **required** |
| `--commit-sha` | Full 40-char commit SHA; empty means HEAD of default branch | unset |
| `--branch` | Branch name; ignored when `--commit-sha` is empty | `main` |
| `--webhook-url` | Webhook endpoint | `http://localhost:7091/webhooks/github` |
| `--webhook-secret` | HMAC signing secret; read from `dev/.env.github` if empty | unset |
| `--database-url` | MySQL DSN | local default |

---

## Troubleshooting

### ✗ Failed to connect to webhook endpoint

- Ensure `mise run dev` is running
- Check ctrl-api is healthy: `curl http://localhost:7091/health`

### ✗ Webhook rejected: invalid signature

- Check `UNKEY_GITHUB_APP_WEBHOOK_SECRET` in `dev/.env.github` matches ctrl-api config

### ✗ Failed to fetch repository ID

- Repository must exist on GitHub and be publicly accessible
- Format: `owner/repo` (no `.git` suffix)

### ✗ Build fails with 404 on private repo

- Make sure `UNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS=false` in `dev/.env.github`
- The GitHub App must be installed on the repository
