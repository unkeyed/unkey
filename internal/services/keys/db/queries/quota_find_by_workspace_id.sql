-- name: FindQuotaByWorkspaceID :one
-- FindQuotaByWorkspaceID returns the quota row for a workspace, used to
-- enforce per-workspace API rate limits on root key requests. NULL
-- limit/duration means unlimited; zero means explicitly blocked.
SELECT *
FROM `quota`
WHERE workspace_id = sqlc.arg('workspace_id');
