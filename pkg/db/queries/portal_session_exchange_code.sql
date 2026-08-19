-- name: ExchangePortalSessionCode :execresult
-- Redeems an exchange code for an access token, in place, on the one row that
-- owns the code.
--
-- Single use is structural rather than a check the caller has to remember: the
-- `access_token_hash IS NULL` predicate plus the UNIQUE index on
-- `exchange_code_hash` mean concurrent redemptions race on a single row and
-- exactly one wins. Callers decide via rowsAffected; zero means the code was
-- unknown, already redeemed, or expired.
UPDATE portal_sessions
SET access_token_hash = sqlc.arg(access_token_hash),
    access_token_created_at = sqlc.arg(access_token_created_at),
    access_token_expires_at = sqlc.arg(access_token_expires_at)
WHERE exchange_code_hash = sqlc.arg(exchange_code_hash)
  AND access_token_hash IS NULL
  AND exchange_code_expires_at > sqlc.arg(now);
