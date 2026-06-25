-- name: UpdateKeyCreditsIncrement :exec
UPDATE `keys`
SET remaining_requests = remaining_requests + sqlc.arg('credits')
WHERE id = ?;