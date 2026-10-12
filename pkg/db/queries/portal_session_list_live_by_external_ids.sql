-- name: ListLivePortalSessionsByExternalIDs :many
-- Loads the revocable sessions for the end users ListLivePortalSessionExternalIDs
-- returned, with the same live predicate. Ordered by external_id, then newest
-- first, so callers can group rows in one pass.
SELECT id, external_id, scopes, access_token_hash, access_token_expires_at, exchange_code_expires_at, created_at FROM portal_sessions
WHERE workspace_id = sqlc.arg('workspace_id')
  AND portal_id = sqlc.arg('portal_id')
  AND external_id IN (sqlc.slice('external_ids'))
  AND revoked_at IS NULL
  AND (
    (access_token_hash IS NOT NULL AND access_token_expires_at > sqlc.arg('access_token_expires_after'))
    OR (access_token_hash IS NULL AND exchange_code_expires_at > sqlc.arg('exchange_code_expires_after'))
  )
ORDER BY external_id ASC, created_at DESC, id ASC;
