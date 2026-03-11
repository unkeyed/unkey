-- name: FindDeploymentById :one
-- FindDeploymentById returns a single deployment by its unique identifier.
-- Used by the router service to look up deployment metadata (sentinel config,
-- environment ownership) when routing an incoming request.
SELECT
  id,
  workspace_id,
  project_id,
  environment_id,
  sentinel_config
FROM `deployments`
WHERE id = sqlc.arg(id);
