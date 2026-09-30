-- name: RevokePortalSessionsByExternalID :execrows
-- Revokes one end user's live sessions on a portal: an unexpired access token,
-- or an unexpired code that was never exchanged. Expired rows are left alone so
-- the count reflects access that was actually cut.
UPDATE portal_sessions
SET revoked_at = sqlc.arg('revoked_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND portal_id = sqlc.arg('portal_id')
  AND external_id = sqlc.arg('external_id')
  AND revoked_at IS NULL
  AND (
    (access_token_hash IS NOT NULL AND access_token_expires_at > sqlc.arg('access_token_expires_after'))
    OR (access_token_hash IS NULL AND exchange_code_expires_at > sqlc.arg('exchange_code_expires_after'))
  );
