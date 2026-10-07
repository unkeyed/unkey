-- name: UpdateUnkeyRootKeysLastUsed :exec
-- UpdateUnkeyRootKeysLastUsed advances last-used time for new root keys without regressing newer values.
UPDATE unkey_root_keys
SET last_used_at = sqlc.arg('last_used_at')
WHERE id IN (sqlc.slice('key_ids'))
  AND last_used_at < sqlc.arg('last_used_at');
