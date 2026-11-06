-- name: FindAppPolicyConfigByID :one
-- Returns the sentinel_config of an app's current deployment, scoped to the
-- workspace. Used by portal.createSession to resolve the keyspaces an
-- app-mapped portal config grants access to (the keyauth policies carry the
-- keySpaceIds verified at the gateway).
SELECT d.sentinel_config
FROM apps a
JOIN environments e ON e.id = (
  SELECT pe.id
  FROM environments pe
  WHERE pe.app_id = a.id
    AND pe.kind = 'production'
    AND pe.deleted_at IS NULL
  ORDER BY pe.pk ASC
  LIMIT 1
)
JOIN deployments d ON e.current_deployment_id = d.id
WHERE a.id = sqlc.arg(app_id)
  AND a.workspace_id = sqlc.arg(workspace_id);
