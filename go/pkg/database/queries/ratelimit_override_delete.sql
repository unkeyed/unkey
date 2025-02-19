-- name: DeleteRatelimitOverride :execresult
UPDATE `ratelimit_overrides`
SET
    deleted_at =  sqlc.arg(now)
WHERE id = sqlc.arg(id);
