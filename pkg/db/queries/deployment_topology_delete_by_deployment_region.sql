-- name: DeleteDeploymentTopologyByDeploymentRegion :exec
DELETE FROM `deployment_topology`
WHERE deployment_id = sqlc.arg(deployment_id)
  AND region_id = sqlc.arg(region_id);
