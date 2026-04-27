# GitHub Dev Tools

Local development tools for setting up and testing GitHub App-triggered deployments.

## Commands

- `go run . dev github setup`: create a GitHub App via manifest flow and write all credentials automatically
- `go run . dev github tunnel`: start an ngrok tunnel and update the GitHub App webhook URL automatically
- `go run . dev github trigger-webhook`: simulate a GitHub push webhook to trigger a deployment

---

## Setup

### Step 1: Create the GitHub App

```bash
go run . dev github setup --app-name my-unkey-dev
```

This opens a browser, walks you through GitHub's App creation UI, then writes:

- `dev/.env.github`: app ID, webhook secret, app name
- `dev/.github-private-key.pem`: private key for ctrl-worker
- `web/apps/dashboard/.github-private-key.pem`: private key for the dashboard
- `web/apps/dashboard/.env`: `GITHUB_APP_ID` and `NEXT_PUBLIC_GITHUB_APP_NAME`

### Step 2: Start the dev environment

```bash
make dev
```

Tilt spins up a `github-tunnel` resource that runs ngrok against ctrl-api and patches the GitHub App's webhook URL to point at the public ngrok address. No manual tunnel step needed.

### Step 3: Seed the database

```bash
go run . dev seed local
```

### Step 4: Trigger a deployment

```bash
go run . dev github trigger-webhook \
  --project local-api \
  --repository owner/repo
```

Omitting `--commit-sha` deploys the HEAD of the repo's default branch (resolved via the GitHub API). Pass `--commit-sha <40-char-sha>` and `--branch <name>` to pin a specific commit.

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

- Ensure `make dev` is running
- Check ctrl-api is healthy: `curl http://localhost:7091/health`

### ✗ Webhook rejected: invalid signature

- Check `UNKEY_GITHUB_APP_WEBHOOK_SECRET` in `dev/.env.github` matches ctrl-api config

### ✗ Failed to fetch repository ID

- Repository must exist on GitHub and be publicly accessible
- Format: `owner/repo` (no `.git` suffix)

### ✗ Build fails with 404 on private repo

- Make sure `UNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS=false` in `dev/.env.github`
- The GitHub App must be installed on the repository
