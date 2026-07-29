-- name: FindAppSentinelConfigByID :one
-- Returns the sentinel_config of an app's current deployment, scoped to the
-- workspace. Used by portal.createSession to resolve the keyspaces an
-- app-mapped portal config grants access to (the keyauth policies carry the
-- keySpaceIds verified at the gateway).
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT d.sentinel_config
FROM apps a
JOIN deployments d ON (a.current_deployment_id COLLATE utf8mb4_0900_ai_ci = d.id AND a.current_deployment_id COLLATE utf8mb4_0900_as_cs = d.id)
WHERE a.id = sqlc.arg(app_id)
  AND a.workspace_id = sqlc.arg(workspace_id);
