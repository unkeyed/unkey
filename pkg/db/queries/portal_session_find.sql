-- name: FindValidPortalSession :one
SELECT * FROM portal_sessions
WHERE id = sqlc.arg(id)
  AND expires_at > sqlc.arg(now);
