-- name: RevokePortalSessionsByIDs :execrows
-- Revokes the sessions LockLivePortalSessionsByExternalID locked. Run it in the
-- same transaction, so the ids are exactly the rows this call revokes.
UPDATE portal_sessions
SET revoked_at = sqlc.arg('revoked_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id IN (sqlc.slice('ids'))
  AND revoked_at IS NULL;
