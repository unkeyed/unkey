-- name: DisableLeakedKey :execrows
UPDATE `keys`
SET enabled = FALSE, updated_at_m = sqlc.arg(now)
WHERE hash = sqlc.arg(hash)
  AND COALESCE(for_workspace_id, workspace_id) = sqlc.arg(workspace_id)
  AND enabled = TRUE
  AND deleted_at_m IS NULL;
