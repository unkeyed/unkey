-- name: FindValidPortalSessionByExchangeCode :one
SELECT * FROM portal_sessions
WHERE exchange_code_hash = sqlc.arg(exchange_code_hash)
  AND access_token_created_at IS NULL
  AND exchange_code_expires_at > sqlc.arg(now);
