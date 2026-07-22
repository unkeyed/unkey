-- name: ListDeploymentEnvAndAppState :many
SELECT
  d.id AS deployment_id,
  p.slug AS project_slug,
  a.slug AS app_slug,
  e.slug AS environment_slug,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN projects p ON p.id = d.project_id
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND d.id IN (sqlc.slice('deployment_ids'));
