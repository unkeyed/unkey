-- name: DisableLeakedUnkeyRootKey :execrows
UPDATE unkey_root_keys
SET enabled = false
WHERE id = sqlc.arg(id)
  AND hash = sqlc.arg(hash)
  AND enabled = true
  AND deleted_at IS NULL;
