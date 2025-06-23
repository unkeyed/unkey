-- name: FindRatelimitOverrideByID :one
SELECT * FROM ratelimit_overrides
WHERE
    workspace_id = sqlc.arg(workspace_id)
    AND id = sqlc.arg(override_id);
