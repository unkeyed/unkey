-- name: ListDeploymentEnvAndAppState :many
SELECT
  d.id AS deployment_id,
  p.slug AS project_slug,
  a.slug AS app_slug,
  e.slug AS environment_slug,
  e.kind AS environment_kind,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN projects p ON d.project_id = p.id
JOIN environments e ON d.environment_id = e.id
JOIN apps a ON d.app_id = a.id
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND d.id IN (sqlc.slice('deployment_ids'));
