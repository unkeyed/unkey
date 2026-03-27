-- name: FindValidPortalSessionToken :one
SELECT * FROM portal_session_tokens
WHERE id = sqlc.arg(id)
  AND exchanged_at IS NULL
  AND expires_at > UNIX_TIMESTAMP(NOW()) * 1000;
