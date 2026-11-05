-- name: RevokeLeakedUnkeyRootKey :execrows
UPDATE unkey_root_keys
SET deleted_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
  AND hash = sqlc.arg(hash)
  AND deleted_at IS NULL;
