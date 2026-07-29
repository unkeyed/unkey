-- name: ListDeploymentEnvAndAppState :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
  d.id AS deployment_id,
  p.slug AS project_slug,
  a.slug AS app_slug,
  e.slug AS environment_slug,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN projects p ON (d.project_id COLLATE utf8mb4_0900_ai_ci = p.id AND d.project_id COLLATE utf8mb4_0900_as_cs = p.id)
JOIN environments e ON (d.environment_id COLLATE utf8mb4_0900_ai_ci = e.id AND d.environment_id COLLATE utf8mb4_0900_as_cs = e.id)
JOIN apps a ON (d.app_id COLLATE utf8mb4_0900_ai_ci = a.id AND d.app_id COLLATE utf8mb4_0900_as_cs = a.id)
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND d.id IN (sqlc.slice('deployment_ids'));
