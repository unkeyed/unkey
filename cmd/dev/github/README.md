# GitHub Webhook Simulator

Simulates GitHub push webhooks for local development and testing of deployment pipelines.

## First Time Setup

Before triggering webhooks, initialize your local database:

```bash
# Run once to create workspace, project, and test data
go run . dev seed local
```

This creates:

- Local workspace (`ws_local`)
- Test project with a project ID
- Preview and production environments
- Root API key for testing
- GitHub repository connection

The seed command outputs the project ID you'll use with `--project-id` below.

## Quick Start

After running `dev seed local` (see above), trigger a deployment:

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

## Prerequisites

1. **Local dev environment running**:

   ```bash
   make dev  # Starts Tilt with ctrl-api
   ```

2. **Public repository**: The repository must be publicly accessible on GitHub (for local dev without authentication)

3. **Project exists**: Create a project in your local Unkey instance first

## Flags

### Required

- `--project-id` - Your Unkey project ID (e.g., `proj_abc123`)
- `--repository` - Full repository name (e.g., `ogzhanolguncu/demo_api`)
- `--commit-sha` - Git commit SHA to deploy (full 40-char SHA)

### Optional

- `--branch` - Branch name (default: `main`)
- `--webhook-url` - Webhook endpoint (default: `http://localhost:7091/webhooks/github`)
- `--webhook-secret` - HMAC signing secret (default: `local-dev-secret`)
- `--database-url` - MySQL DSN (default: `unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true`)

## How It Works

1. **Fetches repository metadata** from GitHub API (repository ID)
2. **Creates database record** in `github_repo_connection` table (with installation_id=1 for local dev)
3. **Builds webhook payload** matching GitHub's push event format
4. **Signs payload** with HMAC-SHA256 (X-Hub-Signature-256 header)
5. **Sends POST request** to ctrl-api webhook endpoint
6. **Ctrl-api processes webhook** and triggers deployment via BuildKit

## Environment Variables

Override defaults with environment variables:

```bash
export UNKEY_GITHUB_APP_WEBHOOK_SECRET=my-custom-secret
export UNKEY_DATABASE_PRIMARY=mysql://user:pass@localhost:3306/db

go run . dev github trigger-webhook \
  --project-id proj_abc123 \
  --repository owner/repo \
  --commit-sha abc123...
```

## Examples

### Basic deployment

```bash
go run . dev github trigger-webhook \
  --project-id proj_2tF8Qr7QHvwp5JN6J4K9vR3sL1M \
  --repository ogzhanolguncu/demo_api \
  --commit-sha d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3
```

### With custom branch

```bash
go run . dev github trigger-webhook \
  --project-id proj_abc123 \
  --repository owner/repo \
  --commit-sha abc123def456 \
  --branch feature/new-api
```

## Troubleshooting

### ✗ Failed to connect to webhook endpoint

- Ensure `make dev` is running
- Check ctrl-api is healthy: `curl http://localhost:7091/health`

### ✗ Webhook rejected: invalid signature

- Check `UNKEY_GITHUB_APP_WEBHOOK_SECRET` matches ctrl-api config
- Default for local dev: `local-dev-secret`

### ✗ Failed to fetch repository ID

- Repository must exist on GitHub
- Repository must be publicly accessible (for local dev)
- Check repository name format: `owner/repo` (no `.git` suffix)

### Build fails in BuildKit

- Ensure repository has a `Dockerfile` in root (or specify path in deployment config)
- Check BuildKit logs: `kubectl logs -n unkey -l app=depot-buildkit`
- Verify commit SHA exists in repository
