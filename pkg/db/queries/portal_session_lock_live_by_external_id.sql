-- name: LockLivePortalSessionsByExternalID :many
-- Locks up to `limit` of one end user's live sessions on a portal: an unexpired
-- access token, or an unexpired code that was never exchanged. Expired rows are
-- left alone so the revoke reports only access it actually cut. Only rows at or
-- below `max_pk` are taken, so a caller revoking in batches stops even while new
-- sessions are being minted. The lock pins exactly the rows
-- RevokePortalSessionsByIDs then revokes.
SELECT pk, id, workspace_id, portal_id, external_id, scopes, exchange_code_hash, exchange_code_expires_at, access_token_hash, access_token_created_at, access_token_expires_at, revoked_at, return_url, created_at FROM portal_sessions
WHERE workspace_id = sqlc.arg('workspace_id')
  AND portal_id = sqlc.arg('portal_id')
  AND external_id = sqlc.arg('external_id')
  AND revoked_at IS NULL
  AND pk <= sqlc.arg('max_pk')
  AND (
    (access_token_hash IS NOT NULL AND access_token_expires_at > sqlc.arg('access_token_expires_after'))
    OR (access_token_hash IS NULL AND exchange_code_expires_at > sqlc.arg('exchange_code_expires_after'))
  )
ORDER BY pk
LIMIT ?
FOR UPDATE;
