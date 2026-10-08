-- name: CountDeploymentTopologiesByDeploymentId :one
-- CountDeploymentTopologiesByDeploymentId counts a deployment's topology rows. Only tests use this query.
SELECT COUNT(*)
FROM `deployment_topology`
WHERE deployment_id = sqlc.arg(deployment_id);
