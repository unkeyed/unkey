-- name: DeleteInstancesNotInDeploymentInventory :exec
-- Removes stale instance rows for deployment ReplicaSets absent from a
-- complete regional inventory. Call DeleteRegionInstances for an empty
-- inventory because NOT IN (NULL) intentionally matches no rows.
DELETE FROM instances
WHERE instances.region_id = sqlc.arg(region_id)
  AND instances.deployment_id NOT IN (sqlc.slice(live_deployment_ids));

-- name: DeleteRegionInstances :exec
DELETE FROM instances
WHERE instances.region_id = sqlc.arg(region_id);
