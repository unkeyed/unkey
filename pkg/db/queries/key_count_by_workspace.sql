-- name: CountKeysByWorkspace :one
-- CountKeysByWorkspace counts every keys row in a workspace. Intended
-- for tests that prove a failed call wrote nothing.
SELECT COUNT(*)
FROM `keys`
WHERE workspace_id = sqlc.arg(workspace_id);
