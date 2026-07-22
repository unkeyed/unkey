-- name: FindPortalConfigByID :one
-- Looks up a portal configuration by id, scoped to the workspace. Used by
-- update/delete to verify the caller owns the config before mutating it, so a
-- config can never be read or mutated across workspace boundaries.
SELECT * FROM portal_configurations
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id);
