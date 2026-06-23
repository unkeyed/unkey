-- Pull requests we are tracking, one row per (repo, number). The Slack channel
-- id is cached here so we never have to page conversations.list to find it.
CREATE TABLE IF NOT EXISTS pull_requests (
    repo             TEXT        NOT NULL,
    number           INTEGER     NOT NULL,
    channel_id       TEXT        NOT NULL,
    channel_name     TEXT        NOT NULL,
    state            TEXT        NOT NULL DEFAULT 'open', -- open | merged | closed
    author_login     TEXT        NOT NULL,
    reviewer_logins  TEXT[]      NOT NULL DEFAULT '{}',
    url              TEXT        NOT NULL DEFAULT '',
    title            TEXT        NOT NULL DEFAULT '',
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reminded_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at      TIMESTAMPTZ,
    PRIMARY KEY (repo, number)
);

-- Reminder query scans open PRs ordered by staleness.
CREATE INDEX IF NOT EXISTS pull_requests_open_idle_idx
    ON pull_requests (last_activity_at)
    WHERE state = 'open' AND archived_at IS NULL;

-- GitHub login -> Slack user id, so reviewers get real @mentions and invites.
CREATE TABLE IF NOT EXISTS user_map (
    github_login  TEXT NOT NULL PRIMARY KEY,
    slack_user_id TEXT NOT NULL
);

-- Idempotency: GitHub retries deliveries, so we record each delivery id once.
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    delivery_id TEXT        NOT NULL PRIMARY KEY,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
