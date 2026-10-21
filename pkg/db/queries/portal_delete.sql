-- name: DeletePortal :execrows
-- Deletes a portal, scoped to the workspace so one workspace can never delete
-- another's. Returns the row count so a concurrent delete that already removed
-- the row is reported as not-found rather than as a second success. Callers
-- must reject the delete while the portal still has domains.
DELETE FROM portals
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id');
