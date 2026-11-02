-- name: FindEnabledRootKeysByHashesForUpdate :many
SELECT id, hash, workspace_id
FROM unkey_root_keys
WHERE hash IN (sqlc.slice('hashes'))
  AND enabled = TRUE
  AND deleted_at IS NULL
ORDER BY hash
FOR UPDATE;
