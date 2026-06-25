-- name: InsertKeyAuth :exec
INSERT INTO key_auth (
    id,
    workspace_id,
    created_at_m,
    default_prefix,
    default_bytes,
    store_encrypted_keys
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    false
);
