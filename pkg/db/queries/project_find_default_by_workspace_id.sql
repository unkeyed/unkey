-- name: FindDefaultProjectByWorkspaceID :one
SELECT id
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND BINARY slug = 'default'
LIMIT 1;
