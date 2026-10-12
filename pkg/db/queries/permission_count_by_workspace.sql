-- name: CountPermissionsByWorkspace :one
-- CountPermissionsByWorkspace counts every permissions row in a workspace. Intended
-- for tests that prove a failed call wrote nothing.
SELECT COUNT(*)
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id);
