-- name: UpdateKeyLastUsed :exec
UPDATE `keys`
SET last_used_at = sqlc.arg('last_used_at')
WHERE id = sqlc.arg('id')
  AND (last_used_at IS NULL OR last_used_at < sqlc.arg('last_used_at'));
