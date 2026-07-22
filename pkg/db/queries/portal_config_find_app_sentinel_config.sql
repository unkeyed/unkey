-- name: FindAppSentinelConfigByID :one
-- Returns the sentinel_config of an app's current deployment, scoped to the
-- workspace. Used by portal.createSession to resolve the keyspaces an
-- app-mapped portal config grants access to (the keyauth policies carry the
-- keySpaceIds verified at the gateway).
SELECT d.sentinel_config
FROM apps a
JOIN deployments d ON d.id = a.current_deployment_id
WHERE a.id = sqlc.arg(app_id)
  AND a.workspace_id = sqlc.arg(workspace_id);
