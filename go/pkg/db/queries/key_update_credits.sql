-- name: UpdateKeyCredits :exec
UPDATE `keys`
SET remaining_requests = CASE
    WHEN sqlc.narg('operation') = 'set' THEN sqlc.narg('credits')
    WHEN sqlc.narg('operation') = 'increment' THEN remaining_requests + sqlc.narg('credits')
    WHEN sqlc.narg('operation') = 'decrement' THEN remaining_requests - sqlc.narg('credits')
END
WHERE id = ?;
