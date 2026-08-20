-- name: FindPortalByKeyspace :one
-- Resolves the portal mapped to a keyspace within a workspace. See
-- portal_find_by_app.sql for why this is workspace-scoped.
SELECT * FROM portals
WHERE key_auth_id = sqlc.arg('key_auth_id')
  AND workspace_id = sqlc.arg('workspace_id')
LIMIT 1;
