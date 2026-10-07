-- name: LockPortalForMint :one
-- Locks the portal row while a session is minted. Disabling, re-pointing, and
-- deleting a portal all write this row before revoking its sessions, so the
-- lock orders a mint before or after them: either the revoke sees the new
-- session, or the mint sees the change and refuses. The mapping columns let the
-- caller check the row still points where its grant was built from.
SELECT id, enabled, key_auth_id, app_id FROM portals
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
FOR UPDATE;
