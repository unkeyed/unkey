# GitHub Dev Tools

Local development tools for setting up and testing GitHub App-triggered deployments.

## Commands

- `go run . dev github setup` — create a GitHub App via manifest flow and write all credentials automatically
- `go run . dev github trigger-webhook` — simulate a GitHub push webhook to trigger a deployment

---

## Setup

### Step 1: Create the GitHub App

```bash
go run . dev github setup --app-name my-unkey-dev
```

This opens a browser, walks you through GitHub's App creation UI, then writes:

- `dev/.env.github` — app ID, webhook secret, app name
- `dev/.github-private-key.pem` — private key for ctrl-worker
- `web/apps/dashboard/.github-private-key.pem` — private key for the dashboard
- `web/apps/dashboard/.env` — `GITHUB_APP_ID` and `NEXT_PUBLIC_GITHUB_APP_NAME`

### Step 2: Start the dev environment

```bash
make dev
```

### Step 3: Seed the database

```bash
go run . dev seed local
```

This outputs a project ID you'll need in the next step.

### Step 4: Trigger a deployment

```bash
go run . dev github trigger-webhook \
  --project-id proj_abc123 \
  --repository owner/repo \
  --commit-sha <full-40-char-sha>
```

---

## Flags

### `setup`

| Flag | Description | Default |
|------|-------------|---------|
| `--app-name` | GitHub App name (must be globally unique) | `unkey-dev` |
| `--webhook-url` | Webhook URL (update in GitHub App settings once you have a tunnel) | `https://example.com/webhooks/github` |
| `--port` | Local callback server port | `9999` |
| `--out-dir` | Where to write `dev/` credentials | `dev` |

### `trigger-webhook`

| Flag | Description | Default |
|------|-------------|---------|
| `--project-id` | Unkey project ID | **required** |
| `--repository` | Full repository name (`owner/repo`) | **required** |
| `--commit-sha` | Full 40-char commit SHA | **required** |
| `--branch` | Branch name | `main` |
| `--webhook-url` | Webhook endpoint | `http://localhost:7091/webhooks/github` |
| `--webhook-secret` | HMAC signing secret | `supersecret` |
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
