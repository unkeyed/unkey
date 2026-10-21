-- name: LockPortalForUpdate :one
-- Exclusively locks a portal for the rest of the transaction. Attaching a domain
-- in ctrl share-locks the same row, so a delete that checks for domains under
-- this lock cannot race a domain being added.
SELECT id FROM portals
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id)
FOR UPDATE;
