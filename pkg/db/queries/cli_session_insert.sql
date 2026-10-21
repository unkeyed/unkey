-- name: InsertCLISession :exec
INSERT INTO cli_sessions (
    id,
    workspace_id,
    user_id,
    permissions,
    device_name,
    refresh_token_hash,
    refresh_family,
    created_at,
    last_used_at,
    expires_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(user_id),
    CAST(sqlc.arg(permissions) AS JSON),
    sqlc.arg(device_name),
    sqlc.arg(refresh_token_hash),
    sqlc.arg(refresh_family),
    sqlc.arg(created_at),
    sqlc.arg(last_used_at),
    sqlc.arg(expires_at)
);
