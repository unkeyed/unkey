-- name: InsertKeySpace :exec
INSERT INTO `key_auth` (
    id,
    workspace_id,
    created_at_m,
    store_encrypted_keys,
    default_prefix,
    default_bytes,
    size_approx,
    size_last_updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
      sqlc.arg(created_at_m),
    sqlc.arg(store_encrypted_keys),
    sqlc.arg(default_prefix),
    sqlc.arg(default_bytes),
    0,
    0
);
