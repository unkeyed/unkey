-- name: FindRunningInstanceRegionsByDeploymentID :many
SELECT DISTINCT region
FROM instances
WHERE deployment_id = sqlc.arg(deploymentId)
  AND status = 'running';
