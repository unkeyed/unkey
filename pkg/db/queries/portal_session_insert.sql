-- name: InsertPortalSession :exec
-- Creates a session in the `pending` state: an exchange code was minted, no
-- access token has been issued yet. Only the code's hash is stored; the code
-- itself is returned to the caller once and never persisted.
INSERT INTO portal_sessions (
    id,
    workspace_id,
    portal_id,
    external_id,
    scopes,
    preview,
    exchange_code_hash,
    exchange_code_expires_at,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(portal_id),
    sqlc.arg(external_id),
    sqlc.arg(scopes),
    sqlc.arg(preview),
    sqlc.arg(exchange_code_hash),
    sqlc.arg(exchange_code_expires_at),
    sqlc.arg(created_at)
);
