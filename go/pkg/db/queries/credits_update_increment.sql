-- name: UpdateCreditIncrement :exec
UPDATE `credits`
SET remaining = remaining + sqlc.arg('credits'),
    updated_at = ?
WHERE id = ?;
