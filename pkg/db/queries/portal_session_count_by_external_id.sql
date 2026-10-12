-- name: CountPortalSessionsByExternalID :one
-- CountPortalSessionsByExternalID counts every session minted for one end user
-- in a workspace, in any state. Intended for tests.
SELECT COUNT(*)
FROM portal_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND external_id = sqlc.arg(external_id);
