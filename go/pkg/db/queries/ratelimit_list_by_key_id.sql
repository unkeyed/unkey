-- name: ListRatelimitsByKeyID :many
SELECT
  id,
  name,
  `limit`,
  duration,
  auto_apply
FROM ratelimits
WHERE key_id = ?;
