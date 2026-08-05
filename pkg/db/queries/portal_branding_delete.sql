-- name: DeletePortalBranding :exec
-- Deletes a portal configuration's branding row. Called alongside
-- DeletePortalConfig since the schema has no cascading delete.
DELETE FROM portal_branding WHERE portal_config_id = sqlc.arg(portal_config_id);
