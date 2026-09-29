-- name: SoftDeleteUnkeyRootKey :execrows
-- SoftDeleteUnkeyRootKey tombstones a live new-format root key in one workspace.
UPDATE unkey_root_keys
SET deleted_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
    AND workspace_id = sqlc.arg(workspace_id)
    AND deleted_at IS NULL;
