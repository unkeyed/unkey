-- name: UpdateRatelimitOverride :execresult
UPDATE `ratelimit_overrides`
SET
    `limit` = sqlc.arg(windowLimit),
    duration = sqlc.arg(duration),
    async = sqlc.arg(async),
    updated_at_m= sqlc.arg(now)
WHERE id = sqlc.arg(id);
