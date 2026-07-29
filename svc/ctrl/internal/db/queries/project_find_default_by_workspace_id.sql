-- name: FindDefaultProjectByWorkspaceID :one
-- FindDefaultProjectByWorkspaceID resolves only the exact lowercase default slug.
-- BINARY prevents case-insensitive collations from accepting a different project.
SELECT id
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND BINARY slug = 'default'
LIMIT 1;
