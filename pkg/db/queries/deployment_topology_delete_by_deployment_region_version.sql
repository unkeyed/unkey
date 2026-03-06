-- name: DeleteDeploymentTopologyByDeploymentRegionVersion :exec
DELETE FROM `deployment_topology`
WHERE deployment_id = sqlc.arg(deployment_id)
  AND region = sqlc.arg(region)
  AND version = sqlc.arg(version);
