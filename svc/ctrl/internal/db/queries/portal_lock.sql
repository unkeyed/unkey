-- name: LockPortal :one
-- Share-locks a tenant's portal while a domain is attached to it. Deleting the
-- portal locks the same row exclusively, so the two serialize and a domain can
-- never be attached to a portal that is mid-delete or already gone.
SELECT id FROM portals
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id)
LOCK IN SHARE MODE;
