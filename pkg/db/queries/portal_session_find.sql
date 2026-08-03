-- name: FindValidPortalSession :one
SELECT portal_sessions.pk, portal_sessions.id, portal_sessions.workspace_id, portal_sessions.portal_config_id, portal_sessions.external_id, portal_sessions.permissions, portal_sessions.preview, portal_sessions.expires_at, portal_sessions.created_at FROM portal_sessions
WHERE id = sqlc.arg(id)
  AND expires_at > sqlc.arg(now);
