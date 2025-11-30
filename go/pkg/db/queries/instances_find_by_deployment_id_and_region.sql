-- name: FindInstancesByDeploymentIdAndRegion :many
SELECT
  id,
  deployment_id,
  workspace_id,
  project_id,
  region,
  address,
  cpu_millicores,
  memory_mib,
  status
FROM instances
WHERE deployment_id = sqlc.arg(deploymentId) AND region = sqlc.arg(region);
