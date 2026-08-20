-- name: DeleteManyRatelimitsByIDs :exec
DELETE FROM ratelimits WHERE id IN (sqlc.slice(ids));

-- name: DeleteAllRatelimitsByKeyID :exec
DELETE FROM ratelimits
WHERE key_id = sqlc.arg(key_id);

-- name: DeleteRatelimitsByKeyIDExceptNames :exec
DELETE FROM ratelimits
WHERE key_id = ?
  AND name NOT IN (sqlc.slice('ratelimit_names'));
