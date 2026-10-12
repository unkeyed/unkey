-- name: CountLivePortalSessionsByPortal :one
-- CountLivePortalSessionsByPortal counts a portal's unrevoked sessions,
-- restricted to one end user when external_id is non-empty. Intended for
-- tests.
SELECT COUNT(*)
FROM portal_sessions
WHERE portal_id = sqlc.arg(portal_id)
  AND revoked_at IS NULL
  AND (sqlc.arg(external_id) = '' OR external_id = sqlc.arg(external_id));
