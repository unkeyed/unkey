-- name: ConsumePortalSessionExchangeCode :execresult
UPDATE portal_sessions
SET
    exchange_code_hash = NULL,
    access_token_hash = sqlc.arg(access_token_hash),
    access_token_created_at = sqlc.arg(access_token_created_at),
    access_token_expires_at = sqlc.arg(access_token_expires_at)
WHERE id = sqlc.arg(id)
  AND exchange_code_hash = sqlc.arg(exchange_code_hash)
  AND access_token_created_at IS NULL
  AND access_token_hash IS NULL
  AND exchange_code_expires_at > sqlc.arg(now);
