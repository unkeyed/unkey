-- name: MaxPortalSessionPkByExternalID :one
-- Returns the highest pk among one end user's sessions on a portal, or 0 when
-- there are none. Read on the primary before revoking in batches, it bounds the
-- revoke to sessions that already exist: pk is assigned at insert, so a session
-- minted while the batches run lands above it.
SELECT CAST(COALESCE(MAX(pk), 0) AS UNSIGNED) AS max_pk FROM portal_sessions
WHERE workspace_id = sqlc.arg('workspace_id')
  AND portal_id = sqlc.arg('portal_id')
  AND external_id = sqlc.arg('external_id');
