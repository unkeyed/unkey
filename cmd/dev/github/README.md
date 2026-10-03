# GitHub Dev Tools

Local development tools for setting up and testing GitHub App-triggered deployments.

## Choose a mode

The relay is optional. Neither production nor local development requires it.

| Mode | Installation return | Webhook delivery |
|------|---------------------|------------------|
| Existing production App | Existing dashboard callback | Existing production receiver |
| Your own development App | Your local dashboard callback | Your ngrok tunnel to ctrl-api |
| Shared development relay | Relay callback, then your registered dashboard | Optional relay pull forwarder |
| No GitHub configuration | GitHub installation unavailable | None; container image deployment still works |

For production and your own development App, leave `GITHUB_INSTALL_RELAY_URL`,
`GITHUB_INSTALL_RELAY_TOKEN`, `GITHUB_INSTALL_RELAY_ADMIN_TOKEN`, and
`GITHUB_INSTALL_RELAY_WEBHOOKS` unset in both the process environment and
dashboard env files. Direct mode does not contact the relay, create relay
cookies, or require relay credentials. Keep production App settings and
webhook delivery unchanged. Never run the development tunnel against the
production App.

Relay installation and relay webhook delivery are separate choices. A relay
URL enables installation through the relay; it does not start webhook polling.
Partial relay settings fail closed instead of silently switching to direct mode.

## Commands

- `mise run unkey -- dev github setup`: create a GitHub App via manifest flow and write all credentials automatically
- `mise run unkey -- dev github tunnel`: start an ngrok tunnel and update the GitHub App webhook URL automatically
- `mise run unkey -- dev github trigger-webhook`: simulate a GitHub push webhook to trigger a deployment
- `mise run unkey -- dev github relay-events`: pull registered events from the installation relay into the local control API

---

## Set up your own development App without a relay

Run the setup command on your machine, where the browser can reach localhost.
The manifest creates a separate App with the permissions and events needed
for deployments. You do not need access to the shared relay or its credentials.

### Step 1: Create the GitHub App

```bash
mise run unkey -- dev github setup --app-name my-unkey-dev
```

This opens a browser, walks you through GitHub's App creation UI, then writes:

- `dev/.env.github`: app ID, webhook secret, app name
- `dev/.github-private-key.pem`: private key for ctrl-worker
- `web/apps/dashboard/.github-private-key.pem`: private key for the dashboard
- `web/apps/dashboard/.env`: `GITHUB_APP_ID`, `NEXT_PUBLIC_GITHUB_APP_NAME`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`

The manifest sets both the first **Callback URL** and **Setup URL** to
`http://localhost:3000/integrations/github/callback`, enables user authorization
during installation and Redirect on update, and subscribes to `push` and
`pull_request`. It requests Contents and Metadata read access, and Deployments,
Commit statuses, and Pull requests write access.

For a dashboard on a different port or a public portal, change both App URLs
to `<dashboard-origin>/integrations/github/callback` and set
`DASHBOARD_BASE_URL` to that origin in `web/apps/dashboard/.env.local`.
Use the browser-accessible public origin for an orb, not its localhost address.
Do not add a trailing slash or enable wildcard callback matching.

For an existing development App, supply the same credentials instead of
running setup. Tilt reads `UNKEY_GITHUB_APP_ID`,
`UNKEY_GITHUB_APP_WEBHOOK_SECRET`, and `NEXT_PUBLIC_GITHUB_APP_NAME` from
`dev/.env.github`, and the App private key from `dev/.github-private-key.pem`.
The dashboard needs `GITHUB_APP_ID`, `NEXT_PUBLIC_GITHUB_APP_NAME`,
`GITHUB_CLIENT_ID`, and `GITHUB_CLIENT_SECRET` in its env file. Tilt injects
the private key as `UNKEY_GITHUB_PRIVATE_KEY_PEM`; supply it yourself if you
run the dashboard without Tilt. Keep these files and secrets out of Git.
OAuth credentials remain required for a new workspace binding, as in the
existing direct flow; the relay does not add that requirement.

### Step 2: Start the dev environment

```bash
mise run dev
```

With the generated credential files present and relay settings unset, Tilt
starts `github-tunnel`. Install and authenticate ngrok before starting Tilt.
The tunnel targets ctrl-api and updates your development App's webhook URL.
The webhook secret comes from `dev/.env.github`. Install the App from the
dashboard's workspace settings and choose the repositories to use.
Relay users must not run this tunnel against the shared App.

### Step 3: Seed the database

```bash
mise run unkey -- dev seed local
```

### Step 4: Trigger a deployment

```bash
mise run unkey -- dev github trigger-webhook \
  --project local-api \
  --repository owner/repo
```

Omitting `--commit-sha` deploys the HEAD of the repo's default branch (resolved via the GitHub API). Pass `--commit-sha <40-char-sha>` and `--branch <name>` to pin a specific commit.

---

## Install through a shared development relay

Use an existing development relay. Obtain its URL and the development App
credentials from its operator; do not change the shared App's settings.
Operating the relay is covered in the
[relay README](https://github.com/unkeyed/github-relay#readme).

Set the dashboard's `DASHBOARD_BASE_URL` to its exact HTTPS origin and
`GITHUB_INSTALL_RELAY_URL` to the relay origin, without trailing slashes.
Local HTTPS works if your browser can reach the dashboard and trusts its
certificate. Include any non-default port. The relay redirects your browser
back; it does not need network access to the dashboard. The dashboard server
must be able to reach the relay.

Supply either a scoped `GITHUB_INSTALL_RELAY_TOKEN` or a
`GITHUB_INSTALL_RELAY_ADMIN_TOKEN`, never both. Admin mode automatically enrolls
the dashboard origin and is for trusted development only. The dashboard still
needs `GITHUB_APP_ID`, `UNKEY_GITHUB_PRIVATE_KEY_PEM`, and
`NEXT_PUBLIC_GITHUB_APP_NAME` for the same App. Relay mode does not need
`GITHUB_CLIENT_ID` or `GITHUB_CLIENT_SECRET` in the dashboard.

Add the following to `web/apps/dashboard/.env.local`, replacing the example
origins and obtaining credentials through your secret manager:

```bash
DASHBOARD_BASE_URL=https://your-dashboard.example.com
GITHUB_INSTALL_RELAY_URL=https://your-relay.example.com
```

For scoped mode, ask the operator to enroll that exact dashboard origin and
deliver `GITHUB_INSTALL_RELAY_TOKEN` privately. For trusted automatic enrollment,
supply `GITHUB_INSTALL_RELAY_ADMIN_TOKEN` privately and leave the scoped token
unset. An admin token can enroll any origin and revoke other registrations;
never expose it to untrusted PRs or use production credentials for previews.
Restart the dashboard, then click **Install GitHub App** in workspace settings.

Also export `GITHUB_INSTALL_RELAY_URL` into the process that starts Tilt, even
if you do not want webhook forwarding. Tilt cannot see settings stored only
in dashboard env files; without the process setting it can start ngrok and
overwrite the shared App's webhook URL.

## Receive events through a shared development relay

If your relay supports webhook delivery, the local forwarder pulls your
environment's queued events and sends them to its loopback control API.
No public local webhook endpoint is required.

1. Obtain the development App's webhook secret from the relay operator.
   Set `UNKEY_GITHUB_APP_WEBHOOK_SECRET` in `dev/.env.github` to that secret
   and restart the local control API after updating its Kubernetes secret.
2. Set `GITHUB_INSTALL_RELAY_URL` and `DASHBOARD_BASE_URL` to the exact HTTPS
   relay and dashboard origins, without trailing slashes. Supply exactly one
   credential through the environment: `GITHUB_INSTALL_RELAY_TOKEN`, or
   `GITHUB_INSTALL_RELAY_ADMIN_TOKEN` for trusted development only. Never give
   the admin token to an untrusted preview.
3. Start the forwarder, then complete the GitHub installation or authorization
   flow from this dashboard. The relay records the repositories that the user
   can access within the installation. A database seed or a caller-provided ID
   does not authorize a subscription. Reconnect every 24 hours to renew that
   proof, and after adding repositories. Enrollment renewal alone does not
   extend repository access.

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

Only `push` and `pull_request` events are forwarded. Old events are not replayed
to newly linked environments. Stop the forwarder to pause consumption; ask the
operator to revoke the environment registration when you no longer need it.
See the relay README for delivery limits and retention.

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
