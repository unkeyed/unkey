-- name: DeleteDeploymentTopologyByDeploymentRegionVersion :exec
DELETE FROM `deployment_topology`
WHERE deployment_id = sqlc.arg(deployment_id)
  AND region_id = (
      SELECT id
      FROM `regions`
      WHERE name = sqlc.arg(region)
      LIMIT 1
  )
  AND version = sqlc.arg(version);
