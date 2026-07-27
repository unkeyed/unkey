-- name: FindDefaultProjectByWorkspaceID :one
-- FindDefaultProjectByWorkspaceID resolves only the exact lowercase default slug.
-- BINARY prevents case-insensitive collations from accepting a different project.
SELECT id
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND BINARY slug = 'default'
LIMIT 1;

-- name: LockDefaultProjectByWorkspaceID :one
-- LockDefaultProjectByWorkspaceID uses a current read so a transaction can
-- observe a default project created after its repeatable-read snapshot.
SELECT id
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND BINARY slug = 'default'
LIMIT 1
FOR UPDATE;
