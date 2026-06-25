-- name: UpdateKeyCreditsSet :exec
UPDATE `keys`
SET remaining_requests = sqlc.narg('credits')
WHERE id = ?;