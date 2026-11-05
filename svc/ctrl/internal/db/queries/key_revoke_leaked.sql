-- name: RevokeLeakedKey :execrows
UPDATE `keys`
SET deleted_at_m = sqlc.arg(now)
WHERE id = sqlc.arg(id)
  AND hash = sqlc.arg(hash)
  AND deleted_at_m IS NULL;
