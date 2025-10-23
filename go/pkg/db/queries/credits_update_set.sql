-- name: UpdateCreditSet :exec
UPDATE `credits`
SET remaining = sqlc.arg('remaining'),
    updated_at = ?
WHERE id = ?;
