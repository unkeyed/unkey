-- name: DisableLeakedRootKeys :exec
UPDATE unkey_root_keys
SET enabled = FALSE
WHERE id IN (sqlc.slice('ids'))
  AND enabled = TRUE
  AND deleted_at IS NULL;
