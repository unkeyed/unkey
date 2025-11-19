-- name: FindInstancesByDeploymentIdAndRegion :many
SELECT
  id,
  deployment_id,
  region,
  address,
  status
FROM instances
WHERE deployment_id = sqlc.arg(deploymentId) AND region = sqlc.arg(region);
