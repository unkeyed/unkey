-- name: GetKeyAuthByID :one
SELECT
    id,
    workspace_id,
    created_at_m,
    default_prefix,
    default_bytes,
    store_encrypted_keys
FROM key_auth
WHERE id = ?
  AND deleted_at_m IS NULL;
