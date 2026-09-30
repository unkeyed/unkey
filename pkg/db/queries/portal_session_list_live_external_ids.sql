-- name: ListLivePortalSessionExternalIDs :many
-- Returns one page of the end users holding a revocable session on a portal,
-- using the same live predicate as LockLivePortalSessionsByExternalID. Ordered
-- by external_id with external_id >= external_id_cursor, so an empty cursor
-- starts at the first end user. search is a LIKE pattern from
-- mysql.SearchPrefix; NULL disables the filter. The hint pins idx_portal_revoked:
-- left to itself the planner can walk the workspace-wide idx_external_id and
-- scan every other portal's sessions to fill a page.
SELECT DISTINCT external_id FROM portal_sessions FORCE INDEX (idx_portal_revoked)
WHERE workspace_id = sqlc.arg('workspace_id')
  AND portal_id = sqlc.arg('portal_id')
  AND revoked_at IS NULL
  AND (
    (access_token_hash IS NOT NULL AND access_token_expires_at > sqlc.arg('access_token_expires_after'))
    OR (access_token_hash IS NULL AND exchange_code_expires_at > sqlc.arg('exchange_code_expires_after'))
  )
  AND external_id >= sqlc.arg('external_id_cursor')
  AND (sqlc.narg('search') IS NULL OR external_id LIKE sqlc.narg('search'))
ORDER BY external_id ASC
LIMIT ?;
