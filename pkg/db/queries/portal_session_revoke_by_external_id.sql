-- name: RevokePortalSessionsByExternalID :execrows
-- Revokes every live session one end user holds on a portal, scoped to the
-- workspace.
--
-- Live means a session that could still authenticate: an access token that has
-- not expired, or a pending exchange code that has not. Expired rows are left
-- untouched so the returned count, and the audit log built from it, reflect
-- access that was actually cut. Pending rows are included so a code issued
-- before the revoke cannot be redeemed after it.
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
