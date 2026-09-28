-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET first_ready_at = COALESCE(first_ready_at, CASE
        WHEN status IN ('ready', 'stopped') THEN COALESCE(updated_at, created_at)
        WHEN sqlc.arg(status) = 'ready' THEN COALESCE(sqlc.narg(updated_at), created_at)
        ELSE NULL
    END),
    status = sqlc.arg(status), updated_at = sqlc.narg(updated_at)
WHERE id = sqlc.arg(id);
