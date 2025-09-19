-- name: RollBackDomain :exec
UPDATE domains
SET deployment_id = sqlc.arg(target_deployment_id),
    is_rolled_back = sqlc.arg(is_rolled_back),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
