-- name: UpdateBuildFailed :exec
UPDATE builds SET 
    status = 'failed',
    completed_at = sqlc.arg(now),
    error_message = sqlc.arg(error_message),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);