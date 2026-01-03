-- name: UpdateRatelimit :exec
UPDATE `ratelimits`
SET
    name = sqlc.arg('name'),
    `limit` = sqlc.arg('limit'),
    duration = sqlc.arg('duration'),
    auto_apply = sqlc.arg('auto_apply'),
    updated_at = NOW()
WHERE
    id = sqlc.arg('id');
