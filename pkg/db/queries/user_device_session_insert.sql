-- name: InsertUserDeviceSession :exec
INSERT INTO user_device_sessions (
    id,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    refresh_family,
    created_at,
    last_used_at,
    expires_at,
    oauth_client_id,
    audience
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(user_id),
    sqlc.arg(user_name),
    CAST(sqlc.arg(permissions) AS JSON),
    sqlc.arg(device_name),
    sqlc.arg(refresh_token_hash),
    sqlc.arg(refresh_family),
    sqlc.arg(created_at),
    sqlc.arg(last_used_at),
    sqlc.arg(expires_at),
    sqlc.arg(oauth_client_id),
    sqlc.arg(audience)
);
