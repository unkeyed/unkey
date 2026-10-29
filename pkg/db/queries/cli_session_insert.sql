-- name: InsertCLISession :exec
INSERT INTO cli_sessions (
    id,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    refresh_family,
    client_id,
    created_at,
    last_used_at,
    expires_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(user_id),
    sqlc.arg(user_name),
    CAST(sqlc.arg(permissions) AS JSON),
    sqlc.arg(device_name),
    sqlc.arg(refresh_token_hash),
    sqlc.arg(refresh_family),
    sqlc.arg(client_id),
    sqlc.arg(created_at),
    sqlc.arg(last_used_at),
    sqlc.arg(expires_at)
);
