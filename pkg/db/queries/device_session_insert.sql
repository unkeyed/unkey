-- name: InsertDeviceSession :exec
INSERT INTO device_sessions (
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    refresh_family,
    created_at,
    last_used_at,
    expires_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(kind),
    sqlc.arg(workspace_id),
    sqlc.arg(user_id),
    sqlc.arg(user_name),
    CAST(sqlc.arg(permissions) AS JSON),
    sqlc.arg(device_name),
    sqlc.arg(oauth_client_id),
    sqlc.arg(resource),
    sqlc.arg(refresh_token_hash),
    sqlc.arg(refresh_family),
    sqlc.arg(created_at),
    sqlc.arg(last_used_at),
    sqlc.arg(expires_at)
);
