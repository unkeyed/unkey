-- name: CountKeyPermissionsByWorkspace :one
-- CountKeyPermissionsByWorkspace counts every keys_permissions row in a workspace. Intended
-- for tests that prove a failed call wrote nothing.
SELECT COUNT(*)
FROM keys_permissions
WHERE workspace_id = sqlc.arg(workspace_id);
