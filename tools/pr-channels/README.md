# pr-channels

A standalone bot that keeps PR reviews from getting lost. It does two things:

- **Overview feed** — posts opened (ready-for-review) and merged PRs to one
  shared channel (`OVERVIEW_CHANNEL_ID`), a high-level activity view. Closes
  without a merge are ignored here.
- **Per-PR review channels** — when a reviewer is requested, the PR gets its own
  public `#pr-<repo>-<number>-<branch>` channel. The author and requested reviewers are
  invited, review/comment/CI activity is posted in, and the channel is archived
  when the PR closes. A PR nobody is asked to review never gets a channel.

This is a self-hosted alternative to Axolo/Pullpo. It runs as a single Go binary
against a small Postgres database (a $5 PlanetScale Postgres is plenty) and is
deployed on Unkey Deploy.

## How it works

```
GitHub App ──webhook──▶ pr-channels (serve) ──Slack Web API──▶ workspace
                              │
                              ▼
                          Postgres
            (channel-id cache · user map · reminders · dedup)
```

Channel names are a pure function of `repo + PR number + head branch` (a PR's
head branch is fixed for its lifetime), so handlers never have to look up or
persist a name to find a PR's channel again. The branch is slugified and only
for readability; `pr-<repo>-<number>` is always preserved within Slack's 80-char
limit. Postgres exists for the things webhooks alone cannot give you: a
channel-id cache (avoids rate-limited `conversations.list`), the GitHub→Slack
user map (real `@mentions`), stale-PR reminders, and webhook idempotency.

### Trigger and scope

- **Overview feed** gets a post when a PR is **ready for review** (opened as a
  non-draft, or a draft marked ready) and again when it is **merged**. Closes
  without a merge and bot PRs (dependabot/renovate) are skipped.
- **A review channel** is created only when there is someone to review:
  - a reviewer is requested (`review_requested`), or
  - the PR is opened/marked-ready with reviewers already attached.

  Drafts and bot PRs get no channel. If a reviewer is requested later, the
  channel is created then; further reviewers are invited into the existing one.
- By default every repo in the org is managed. Set `REPO_ALLOWLIST` to a
  comma-separated list of repo names to narrow it.

### What gets posted (Phase 1)

One-way GitHub → Slack, high-signal only.

Overview feed:

- PR opened (ready for review), with a link to its review channel if one exists
- PR merged

Per-PR review channel:

- PR summary on open (title, author, diffstat, reviewers)
- review requests (newly invited reviewer)
- review submissions (approved / changes requested / commented, with the body)
- top-level PR comments
- CI **failures** (`check_suite` non-success)
- merged / closed outcome, then archive

Inline diff comments and two-way sync are intentionally out of scope — they make
PRs noisy and are the expensive part. They can come later if the team wants them.

## Running

It is a single server. Migrations run automatically on boot.

```bash
mise run pr-channels    # start the server (loads .env)
```

For the dev loop, run from this directory:

```bash
cd tools/pr-channels
go test ./...
```

## Configuration

Copy [`.env.example`](./.env.example) to `.env` and fill it in; the
`mise run pr-channels` task loads it automatically.

| Env | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | yes | PlanetScale Postgres (pgx connection string) |
| `GITHUB_WEBHOOK_SECRET` | yes | Verifies webhook signatures |
| `SLACK_BOT_TOKEN` | yes | Slack bot token (`xoxb-...`) |
| `REPO_ALLOWLIST` | no | Repo names to manage; empty/`*` = whole org |
| `OVERVIEW_CHANNEL_ID` | no | Overview feed channel ID for opened/merged PRs; empty disables the feed |
| `PORT` | no | Listen port (default 8080; Unkey Deploy sets it) |
| `ARCHIVE_DELAY` | no | Delay before archiving a closed PR's channel (default `1m`) |
| `REMINDER_IDLE` | no | Idle time before a PR is nudged (default `18h`) |
| `REMINDER_INTERVAL` | no | How often the reminder loop runs (default `15m`) |

## Setup

### Slack app

Create a Slack app, add these **bot** scopes, install to the workspace, copy the
bot token into `SLACK_BOT_TOKEN`:

- `channels:manage` — create/archive public channels and invite users
- `chat:write` — post messages
- `reactions:write` — emoji reactions
- `users:read`, `users:read.email` — resolve the GitHub→Slack user map

### GitHub App

Create an org-owned GitHub App, set the webhook URL to
`https://<deploy-url>/webhook/github`, set the webhook secret to
`GITHUB_WEBHOOK_SECRET`, and subscribe to events:

- Pull requests
- Pull request reviews
- Issue comments
- Check suites

Install it on the org (all repos, or the ones you want). No per-repo workflow
files are needed.

### User map

Reviewers only get invited and `@mentioned` if we can resolve their GitHub login
to a Slack id. Seed the `user_map` table by hand:

```sql
INSERT INTO user_map (github_login, slack_user_id) VALUES ('chronark', 'U123ABC');
```

Unmapped users still appear, as `` `@login` `` plain text, just without a ping.

### Database queries

SQL lives in [`internal/store/queries`](./internal/store/queries) and is
compiled to typed Go by [sqlc](https://sqlc.dev) (engine `postgresql`, driver
`pgx/v5`); the migrations double as the schema. After editing a `.sql` file,
regenerate the `*_generated.go` code:

```bash
go generate ./internal/store/
```

The methods in `store.go` are thin domain wrappers over the generated `Queries`.

## Endpoints

- `GET /health` — liveness probe
- `POST /webhook/github` — GitHub webhook receiver (signature-verified)

## Reminders

The server runs an internal loop every `REMINDER_INTERVAL` that nudges reviewers
on PRs idle longer than `REMINDER_IDLE`. Across replicas, a Postgres lease lock
(`cron_locks`) ensures only one instance fires per interval. A PR is nudged at
most once per idle window, tracked via `reminded_at`, so even a double-fire is
harmless.

## Notes and limits

- **Channel sprawl** is the real failure mode. Archived channels still count
  toward the workspace channel cap. The `pr-` prefix keeps them filterable, and
  skipping drafts/bots cuts most of the volume.
- **Slack rate limits** tightened in 2025. The bot avoids history polling; the
  Slack client (slack-go) honours `Retry-After` on 429s.
- Webhook handling is **idempotent** on `X-GitHub-Delivery`, so GitHub retries
  are safe.
