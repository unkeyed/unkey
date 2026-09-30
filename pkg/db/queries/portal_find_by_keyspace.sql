-- name: FindPortalByKeyspace :one
-- Resolves the portal mapped to a keyspace within a workspace. See
-- portal_find_by_app.sql for why this is workspace-scoped.
SELECT pk, id, workspace_id, project_id, slug, display_name, app_id, key_auth_id, enabled, logo_url, primary_color, created_at, updated_at FROM portals
WHERE key_auth_id = sqlc.arg('key_auth_id')
  AND workspace_id = sqlc.arg('workspace_id')
LIMIT 1;
