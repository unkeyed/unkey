-- name: DeletePortal :exec
-- Deletes a portal, scoped to the workspace so one workspace can never delete
-- another's. Branding lives on the portal row, so there is no side table to
-- clean up.
DELETE FROM portals
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id');
