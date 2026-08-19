-- name: LockRoleByIDAndWorkspaceID :one
SELECT id, name
FROM roles
WHERE id = sqlc.arg(role_id) AND workspace_id = sqlc.arg(workspace_id)
FOR UPDATE;
