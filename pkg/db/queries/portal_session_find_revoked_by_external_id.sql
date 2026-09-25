-- name: FindPortalSessionsRevokedAtByExternalID :many
-- Reads back the rows RevokePortalSessionsByExternalID just revoked, matched by
-- the exact revoked_at it wrote, so the caller can write their revoked state
-- into the session cache. Run it on the same transaction as the revoke: it then
-- returns exactly the rows that update touched.
SELECT pk, id, workspace_id, portal_id, external_id, scopes, exchange_code_hash, exchange_code_expires_at, access_token_hash, access_token_created_at, access_token_expires_at, revoked_at, return_url, created_at FROM portal_sessions
WHERE workspace_id = sqlc.arg('workspace_id')
  AND portal_id = sqlc.arg('portal_id')
  AND external_id = sqlc.arg('external_id')
  AND revoked_at = sqlc.arg('revoked_at');
