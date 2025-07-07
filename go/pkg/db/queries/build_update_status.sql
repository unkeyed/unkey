-- name: UpdateBuildStatus :exec
UPDATE builds SET 
    status = sqlc.arg(status),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
