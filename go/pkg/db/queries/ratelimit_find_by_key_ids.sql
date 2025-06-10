-- name: FindRatelimitsByKeyIDs :many
SELECT 
  id,
  key_id,
  name,
  `limit`,
  duration
FROM ratelimits 
WHERE key_id IN (sqlc.slice(key_ids))
ORDER BY key_id, id;