-- name: RevokePortalSessionsByPortal :execrows
-- Revokes every live session belonging to a portal, scoped to the workspace.
--
-- A session's keyspace scope is frozen in `scopes` at mint time and the session
-- resolver never reads `portals`, so deleting a portal or re-pointing its
-- association would otherwise leave end users authenticated against stale scope
-- until their access token expired. Returns the row count so the caller can
-- record how many sessions were revoked.
UPDATE portal_sessions
SET revoked_at = sqlc.arg('revoked_at')
WHERE portal_id = sqlc.arg('portal_id')
  AND workspace_id = sqlc.arg('workspace_id')
  AND revoked_at IS NULL;
