-- name: InsertPortalSession :exec
INSERT INTO portal_sessions (
    id,
    workspace_id,
    portal_id,
    external_id,
    permissions,
    exchange_code_hash,
    exchange_code_expires_at,
    access_token_hash,
    access_token_created_at,
    access_token_expires_at,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(portal_id),
    sqlc.arg(external_id),
    sqlc.arg(permissions),
    sqlc.arg(exchange_code_hash),
    sqlc.arg(exchange_code_expires_at),
    sqlc.arg(access_token_hash),
    sqlc.arg(access_token_created_at),
    sqlc.arg(access_token_expires_at),
    sqlc.arg(created_at)
);
