-- name: UpdateKeyCreditsDecrement :exec
UPDATE `keys`
SET remaining_requests = CASE
    WHEN remaining_requests >= sqlc.arg('credits') THEN remaining_requests - sqlc.arg('credits')
    ELSE 0
END
WHERE id = ?;
