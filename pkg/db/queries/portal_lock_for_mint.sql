-- name: LockPortalForMint :one
-- Locks the portal row while a session is minted. Disabling, re-pointing, and
-- deleting a portal all write this row before revoking its sessions, so the
-- lock orders a mint before or after them: either the revoke sees the new
-- session, or the mint sees the change and refuses.
SELECT id, enabled FROM portals
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
FOR UPDATE;
