-- name: UpdateKeysLastUsed :exec
UPDATE `keys`
SET last_used_at = sqlc.arg('last_used_at')
WHERE id IN (sqlc.slice('key_ids'))
  AND last_used_at < sqlc.arg('last_used_at');
