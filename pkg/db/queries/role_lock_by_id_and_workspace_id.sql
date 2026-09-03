-- name: LockRoleByIDAndWorkspaceID :one
-- LockRoleByIDAndWorkspaceID serializes role permission changes. It returns
-- the project ID so authorization can use the role's project-scoped URN.
SELECT id, project_id, name
FROM roles
WHERE id = sqlc.arg(role_id) AND workspace_id = sqlc.arg(workspace_id)
FOR UPDATE;
