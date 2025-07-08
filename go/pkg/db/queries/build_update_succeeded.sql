-- name: UpdateBuildSucceeded :exec
UPDATE builds SET 
    status = 'succeeded',
    completed_at = sqlc.arg(now),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);