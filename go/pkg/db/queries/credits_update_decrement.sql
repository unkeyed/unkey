-- name: UpdateCreditDecrement :exec
UPDATE `credits`
SET remaining = CASE
    WHEN remaining >= sqlc.arg('credits') THEN remaining - sqlc.arg('credits')
    ELSE 0
END,
    updated_at = ?
WHERE id = sqlc.arg('id');
