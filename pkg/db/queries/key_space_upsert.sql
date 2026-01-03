-- name: UpsertKeySpace :exec
INSERT INTO key_auth (
    id,
    workspace_id,
    created_at_m,
    default_prefix,
    default_bytes,
    store_encrypted_keys
) VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    store_encrypted_keys = VALUES(store_encrypted_keys);
