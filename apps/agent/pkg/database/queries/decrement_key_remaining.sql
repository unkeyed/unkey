-- name: DecrementKeyRemaining :exec
UPDATE
    `keys`
SET
    remaining_requests = remaining_requests - 1
WHERE
    id = sqlc.arg("id");