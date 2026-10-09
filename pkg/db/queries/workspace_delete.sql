-- name: DeleteWorkspace :exec
-- DeleteWorkspace hard deletes a workspace row without touching its
-- resources. Intended for tests that reset reusable fixtures.
DELETE FROM workspaces
WHERE id = sqlc.arg(id);
