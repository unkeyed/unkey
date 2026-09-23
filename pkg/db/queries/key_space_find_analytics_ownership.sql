-- name: FindKeySpaceAnalyticsOwnership :many
-- FindKeySpaceAnalyticsOwnership resolves candidate keyspaces to their owning projects before analytics authorization.
-- Rows remain available after soft deletion because historical analytics still reference them.
SELECT id, project_id
FROM key_auth
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    id IN (sqlc.slice(key_space_ids))
    OR project_id IN (sqlc.slice(project_ids))
  );
