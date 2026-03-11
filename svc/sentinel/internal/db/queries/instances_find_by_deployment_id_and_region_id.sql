-- name: FindInstancesByDeploymentIdAndRegionID :many
-- FindInstancesByDeploymentIdAndRegionID returns instances for a deployment in
-- a specific region and status.
SELECT
  id,
  address,
  status
FROM instances
WHERE deployment_id = sqlc.arg(deployment_id)
  AND region_id = sqlc.arg(region_id)
  AND status = sqlc.arg(status);
