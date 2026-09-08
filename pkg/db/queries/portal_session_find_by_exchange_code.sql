-- name: FindPortalSessionByExchangeCodeHash :one
-- Reads back the row a redemption just claimed, to build the response and the
-- audit log. Safe to run unconditionally after ExchangePortalSessionCode
-- reported one affected row: the hash is UNIQUE, so this is the same row, and
-- the caller already established it won the race.
SELECT pk, id, workspace_id, portal_id, external_id, scopes, preview, exchange_code_hash, exchange_code_expires_at, access_token_hash, access_token_created_at, access_token_expires_at, revoked_at, return_url, created_at FROM portal_sessions
WHERE exchange_code_hash = sqlc.arg(exchange_code_hash);
