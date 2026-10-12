-- name: CountLogdrainsByWorkspace :one
-- Counts all configured drains, including paused drains, against the allowance.
-- This ordinary read does not serialize concurrent creates; capacity may overshoot.
SELECT COUNT(*) FROM logdrains WHERE workspace_id = sqlc.arg(workspace_id);
