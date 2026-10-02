-- name: CountLogdrainsByWorkspace :one
-- Covered by workspace_id_idx
SELECT COUNT(*)
FROM logdrains
WHERE workspace_id = sqlc.arg(workspace_id);
