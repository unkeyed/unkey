-- name: DeleteDeploymentTopologyByDeploymentId :exec
DELETE FROM `deployment_topology`
WHERE deployment_id = sqlc.arg(deployment_id);
