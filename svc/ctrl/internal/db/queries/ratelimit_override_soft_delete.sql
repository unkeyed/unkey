-- name: SoftDeleteRatelimitOverride :exec
UPDATE `ratelimit_overrides`
SET
    deleted_at_m =  sqlc.arg(now)
WHERE id = sqlc.arg(id);
