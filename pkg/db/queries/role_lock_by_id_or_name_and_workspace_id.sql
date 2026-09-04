-- name: LockRoleByIDOrNameAndWorkspaceID :one
SELECT id, project_id, name
FROM roles
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (id = sqlc.arg('search') OR name = sqlc.arg('search'))
FOR UPDATE;
