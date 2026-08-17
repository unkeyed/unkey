-- name: FindLimitsByWorkspaceID :one
SELECT *
FROM `limits`
WHERE workspace_id = sqlc.arg('workspace_id');
