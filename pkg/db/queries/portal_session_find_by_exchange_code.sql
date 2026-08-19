-- name: FindPortalSessionByExchangeCodeHash :one
-- Reads back the row a redemption just claimed, to build the response and the
-- audit log. Safe to run unconditionally after ExchangePortalSessionCode
-- reported one affected row: the hash is UNIQUE, so this is the same row, and
-- the caller already established it won the race.
SELECT * FROM portal_sessions
WHERE exchange_code_hash = sqlc.arg(exchange_code_hash);
