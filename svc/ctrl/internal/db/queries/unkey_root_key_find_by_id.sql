-- name: FindUnkeyRootKeyByID :one
-- FindUnkeyRootKeyByID reads a root key's hash and deletion state, including
-- soft-deleted keys.
SELECT id, workspace_id, hash, deleted_at
FROM unkey_root_keys
WHERE id = sqlc.arg(id);
