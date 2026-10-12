-- name: CountIdentitiesByWorkspace :one
-- CountIdentitiesByWorkspace counts every identity row in a workspace,
-- including soft-deleted ones. Intended for tests.
SELECT COUNT(*)
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id);
