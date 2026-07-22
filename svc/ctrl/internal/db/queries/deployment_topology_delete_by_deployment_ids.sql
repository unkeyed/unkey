-- name: DeleteDeploymentTopologiesByDeploymentIds :exec
-- DeleteDeploymentTopologiesByDeploymentIds removes desired regional state
-- before its deployments are hard-deleted by the retention sweep.
DELETE FROM deployment_topology
WHERE deployment_id IN (sqlc.slice('deployment_ids'));
