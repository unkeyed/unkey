-- name: DisableLeakedKey :execrows
UPDATE `keys`
SET enabled = false, updated_at_m = sqlc.arg(now)
WHERE id = sqlc.arg(id)
  AND hash = sqlc.arg(hash)
  AND enabled = true
  AND deleted_at_m IS NULL;
