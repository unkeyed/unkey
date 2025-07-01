-- name: ListRatelimitsByKeyID :many
SELECT
  id,
  name,
  `limit`,
  duration
FROM ratelimits
WHERE key_id = ?;
