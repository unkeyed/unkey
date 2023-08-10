-- name: UpdateKey :exec
UPDATE
    `keys`
SET
    hash = sqlc.arg("hash"),
    start = sqlc.arg("start"),
    owner_id = sqlc.arg("owner_id"),
    meta = sqlc.arg("meta"),
    created_at = sqlc.arg("created_at"),
    expires = sqlc.arg("expires"),
    ratelimit_type = sqlc.arg("ratelimit_type"),
    ratelimit_limit = sqlc.arg("ratelimit_limit"),
    ratelimit_refill_rate = sqlc.arg("ratelimit_refill_rate"),
    ratelimit_refill_interval = sqlc.arg("ratelimit_refill_interval"),
    name = sqlc.arg("name"),
    remaining_requests = sqlc.arg("remaining_requests")
WHERE
    id = sqlc.arg("id")