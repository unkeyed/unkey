# GitHub Webhook Simulator

Simulates GitHub push webhooks for local development and testing of deployment pipelines.

## Prerequisites

1. **Local dev environment running**:

   ```bash
   make dev  # Starts Tilt with ctrl-api
   ```

2. **Configure webhook secret** in `dev/.env.github`:

   ```bash
   UNKEY_GITHUB_APP_WEBHOOK_SECRET=supersecret
   ```

   This enables the webhook endpoint in ctrl-api.

3. **Public repository**: The repository must be publicly accessible on GitHub (for local dev without authentication)

## Quick Start

### Step 1: Seed Database

Initialize your local database with test data:

```bash
go run . dev seed local
```

This creates:

- Local workspace (`ws_local`)
- Test project with a project ID
- Preview and production environments
- Root API key for testing
- GitHub repository connection

The seed command outputs the project ID you'll use in the next step.

### Step 2: Trigger Webhook

```bash
# Get any commit SHA from GitHub (e.g., https://github.com/ogzhanolguncu/demo_api/commits)
# Or use git rev-parse if you have the repo locally
COMMIT_SHA=abc123def456...

# Trigger deployment
go run . dev github trigger-webhook \
  --project-id proj_abc123 \
  --repository ogzhanolguncu/demo_api \
  --commit-sha $COMMIT_SHA
```

## Flags

### Required

- `--project-id` - Your Unkey project ID (e.g., `proj_abc123`)
- `--repository` - Full repository name (e.g., `ogzhanolguncu/demo_api`)
- `--commit-sha` - Git commit SHA to deploy (full 40-char SHA)

### Optional

- `--branch` - Branch name (default: `main`)
- `--webhook-url` - Webhook endpoint (default: `http://localhost:7091/webhooks/github`)
- `--webhook-secret` - HMAC signing secret (default: `supersecret`)
- `--database-url` - MySQL DSN (default: `unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true`)

## How It Works

1. **Fetches repository metadata** from GitHub API (repository ID)
2. **Creates database record** in `github_repo_connections` table (with installation_id=1 for local dev)
3. **Builds webhook payload** matching GitHub's push event format
4. **Signs payload** with HMAC-SHA256 (X-Hub-Signature-256 header)
5. **Sends POST request** to ctrl-api webhook endpoint
6. **Ctrl-api processes webhook** and triggers deployment via BuildKit

## Examples

### Basic deployment

```bash
go run . dev github trigger-webhook \
  --project-id proj_2tF8Qr7QHvwp5JN6J4K9vR3sL1M \
  --repository ogzhanolguncu/demo_api \
  --commit-sha d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3
```

## Troubleshooting

### ✗ Failed to connect to webhook endpoint

- Ensure `make dev` is running
- Check ctrl-api is healthy: `curl http://localhost:7091/health`

### ✗ Webhook rejected: invalid signature

- Check `UNKEY_GITHUB_APP_WEBHOOK_SECRET` matches ctrl-api config
- Default for local dev: `supersecret`

### ✗ Failed to fetch repository ID

- Repository must exist on GitHub
- Repository must be publicly accessible (for local dev)
- Check repository name format: `owner/repo` (no `.git` suffix)

### ✗ Commit SHA not found / Build fails

- Verify the commit SHA exists in the repository
- Check GitHub commits page: `https://github.com/owner/repo/commits`
- Copy full 40-character SHA from GitHub UI
