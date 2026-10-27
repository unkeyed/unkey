-- name: DisableLeakedRootKey :execrows
UPDATE unkey_root_keys
SET enabled = FALSE
WHERE hash = sqlc.arg(hash)
  AND workspace_id = sqlc.arg(workspace_id)
  AND enabled = TRUE
  AND deleted_at IS NULL;
