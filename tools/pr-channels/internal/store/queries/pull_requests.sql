-- name: UpsertPR :exec
-- UpsertPR records or refreshes a tracked PR, preserving created_at and
-- clearing any previous archive marker.
INSERT INTO pull_requests
    (repo, number, channel_id, channel_name, state, author_login, reviewer_logins, url, title, last_activity_at)
VALUES (
    sqlc.arg(repo), sqlc.arg(number), sqlc.arg(channel_id), sqlc.arg(channel_name),
    sqlc.arg(state), sqlc.arg(author_login), sqlc.arg(reviewer_logins),
    sqlc.arg(url), sqlc.arg(title), now()
)
ON CONFLICT (repo, number) DO UPDATE SET
    channel_id = EXCLUDED.channel_id,
    channel_name = EXCLUDED.channel_name,
    state = EXCLUDED.state,
    reviewer_logins = EXCLUDED.reviewer_logins,
    url = EXCLUDED.url,
    title = EXCLUDED.title,
    last_activity_at = now(),
    archived_at = NULL;

-- name: AddReviewer :exec
-- AddReviewer appends a reviewer to a tracked PR (deduplicated) and bumps
-- activity.
UPDATE pull_requests
SET reviewer_logins = (
        SELECT array_agg(DISTINCT x)
        FROM unnest(reviewer_logins || sqlc.arg(reviewers)::text[]) AS x
    ),
    last_activity_at = now()
WHERE repo = sqlc.arg(repo) AND number = sqlc.arg(number);

-- name: GetPR :one
-- GetPR returns the tracked PR for a (repo, number).
SELECT repo, number, channel_id, channel_name, state, author_login, reviewer_logins, url, title
FROM pull_requests
WHERE repo = sqlc.arg(repo) AND number = sqlc.arg(number);

-- name: TouchPR :exec
-- TouchPR bumps last_activity_at so the PR is not flagged as stale.
UPDATE pull_requests
SET last_activity_at = now()
WHERE repo = sqlc.arg(repo) AND number = sqlc.arg(number);

-- name: SetPRClosed :exec
-- SetPRClosed records the terminal state (merged or closed) of a PR.
UPDATE pull_requests
SET state = sqlc.arg(state), last_activity_at = now()
WHERE repo = sqlc.arg(repo) AND number = sqlc.arg(number);

-- name: MarkPRArchived :exec
-- MarkPRArchived records that the Slack channel has been archived.
UPDATE pull_requests
SET archived_at = now()
WHERE repo = sqlc.arg(repo) AND number = sqlc.arg(number);

-- name: StaleOpenPRs :many
-- StaleOpenPRs returns open, un-archived PRs whose last activity is older than
-- the cutoff and that have not been reminded since then.
SELECT repo, number, channel_id, author_login, reviewer_logins
FROM pull_requests
WHERE state = 'open' AND archived_at IS NULL
  AND last_activity_at < sqlc.arg(cutoff)
  AND (reminded_at IS NULL OR reminded_at < last_activity_at)
ORDER BY last_activity_at ASC;

-- name: MarkPRReminded :exec
-- MarkPRReminded records that a reminder was sent for a PR.
UPDATE pull_requests
SET reminded_at = now()
WHERE repo = sqlc.arg(repo) AND number = sqlc.arg(number);
