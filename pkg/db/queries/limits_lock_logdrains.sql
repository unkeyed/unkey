-- name: LockLogdrainLimit :one
-- Serialize log drain creation against the unique workspace limits row, including
-- when the workspace has no drains. Support-granted allowances are authoritative.
SELECT logdrains_max FROM `limits` WHERE workspace_id = sqlc.arg(workspace_id) FOR UPDATE;
