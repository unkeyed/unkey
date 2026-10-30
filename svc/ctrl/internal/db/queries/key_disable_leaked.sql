-- name: DisableLeakedKeys :exec
UPDATE `keys`
SET enabled = FALSE, updated_at_m = sqlc.arg(now)
WHERE id IN (sqlc.slice('ids'))
  AND enabled = TRUE
  AND deleted_at_m IS NULL;
