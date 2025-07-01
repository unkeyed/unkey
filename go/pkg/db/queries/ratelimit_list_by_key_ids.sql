-- name: ListRatelimitsByKeyIDs :many
SELECT
  id,
  key_id,
  name,
  `limit`,
  duration,
  auto_apply
FROM ratelimits
WHERE key_id IN (sqlc.slice(key_ids))
ORDER BY key_id, id;
