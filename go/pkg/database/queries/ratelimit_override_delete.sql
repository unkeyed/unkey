-- name: DeleteRatelimitOverride :execresult
UPDATE `ratelimit_overrides`
SET
    deleted_at = NOW()
WHERE id = sqlc.arg(id);
