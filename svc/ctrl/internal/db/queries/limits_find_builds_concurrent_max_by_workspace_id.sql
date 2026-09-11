-- name: FindBuildsConcurrentMaxByWorkspaceID :one
SELECT builds_concurrent_max
FROM `limits`
WHERE workspace_id = sqlc.arg('workspace_id');
