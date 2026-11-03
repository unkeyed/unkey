-- name: ListDeploymentEnvAndAppState :many
SELECT
  d.id AS deployment_id,
  p.slug AS project_slug,
  a.slug AS app_slug,
  e.slug AS environment_slug,
  e.kind AS environment_kind,
  e.current_deployment_id AS environment_current_deployment_id,
  e.is_rolled_back AS environment_is_rolled_back
FROM deployments d
JOIN projects p ON d.project_id = p.id
JOIN environments e ON d.environment_id = e.id
JOIN apps a ON d.app_id = a.id
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND d.id IN (sqlc.slice('deployment_ids'));
