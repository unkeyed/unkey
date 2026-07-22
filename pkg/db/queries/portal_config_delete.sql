-- name: DeletePortalConfig :exec
-- Deletes a portal configuration, scoped to the workspace. Branding must be
-- deleted separately (see DeletePortalBranding): the schema has no cascading
-- delete.
DELETE FROM portal_configurations
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id);
