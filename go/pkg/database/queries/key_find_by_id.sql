-- name: FindRatelimitOverrideByIdentifier :one
SELECT * FROM `ratelimit_overrides`
WHERE identifier = sqlc.arg(identifier);
