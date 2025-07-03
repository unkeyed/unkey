-- name: UpdateKeyCredits :exec
UPDATE `keys`
SET remaining_requests = CASE
    WHEN sqlc.narg('operation') = 'set' THEN sqlc.narg('credits')
    WHEN sqlc.narg('operation') = 'increment' THEN remaining_requests + sqlc.narg('credits')
    WHEN sqlc.narg('operation') = 'decrement' THEN remaining_requests - sqlc.narg('credits')
    WHEN sqlc.narg('operation') = 'decrement' AND remaining_requests- sqlc.arg('credits') > 0 THEN remaining_requests - sqlc.narg('credits')
    WHEN sqlc.narg('operation') = 'decrement' AND remaining_requests - sqlc.arg('credits') <= 0 THEN 0
END
WHERE id = ?;
