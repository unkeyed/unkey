-- name: FindPortalByApp :one
-- Resolves the portal mapped to an app within a workspace. Callers that reach a
-- portal through its app (rather than through an id or slug) use this.
--
-- Workspace-scoped on purpose: `idx_app_id` is unique across the whole table, so
-- an unscoped lookup would return another workspace's portal.
SELECT pk, id, workspace_id, project_id, slug, display_name, app_id, key_auth_id, enabled, logo_url, primary_color, created_at, updated_at FROM portals
WHERE app_id = sqlc.arg('app_id')
  AND workspace_id = sqlc.arg('workspace_id')
LIMIT 1;
