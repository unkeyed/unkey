-- name: ExchangePortalSessionToken :execresult
UPDATE portal_session_tokens
SET exchanged_at = sqlc.arg(exchanged_at)
WHERE id = sqlc.arg(id)
  AND exchanged_at IS NULL
  AND expires_at > UNIX_TIMESTAMP(NOW()) * 1000;
