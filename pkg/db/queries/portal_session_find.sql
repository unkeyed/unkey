-- name: FindValidPortalSession :one
SELECT * FROM portal_sessions
WHERE access_token_hash = sqlc.arg(access_token_hash)
  AND access_token_expires_at > sqlc.arg(now);
