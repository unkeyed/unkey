-- name: FindLimitsByWorkspaceID :one
-- FindLimitsByWorkspaceID returns the limits row for a workspace, used to
-- enforce per-workspace API rate limits on root key requests. NULL
-- api_requests_count_max_per_minute means unlimited; zero means explicitly
-- blocked.
SELECT *
FROM `limits`
WHERE workspace_id = sqlc.arg('workspace_id');
