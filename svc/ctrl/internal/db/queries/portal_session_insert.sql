-- name: InsertPortalSession :exec
-- Inserts a portal session for an end user. Used only by tests; the portal
-- owns session creation.
INSERT INTO portal_sessions (id, workspace_id, portal_id, external_id, scopes, exchange_code_hash, exchange_code_expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(portal_id), sqlc.arg(external_id), sqlc.arg(scopes), sqlc.arg(exchange_code_hash), sqlc.arg(exchange_code_expires_at), sqlc.arg(created_at));
