-- name: CountUnkeyPermissionsByWorkspace :one
-- CountUnkeyPermissionsByWorkspace counts every unkey_principal_permissions row in a workspace. Intended
-- for tests that prove a failed call wrote nothing.
SELECT COUNT(*)
FROM unkey_principal_permissions
WHERE workspace_id = sqlc.arg(workspace_id);
