-- name: FindValidPortalSessionToken :one
SELECT portal_session_tokens.pk, portal_session_tokens.id, portal_session_tokens.workspace_id, portal_session_tokens.portal_config_id, portal_session_tokens.external_id, portal_session_tokens.permissions, portal_session_tokens.preview, portal_session_tokens.exchanged_at, portal_session_tokens.expires_at, portal_session_tokens.created_at FROM portal_session_tokens
WHERE id = sqlc.arg(id)
  AND exchanged_at IS NULL
  AND expires_at > sqlc.arg(now);
