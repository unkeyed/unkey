-- name: FindDeploymentTopologyByIDAndRegion :one
SELECT
    d.id,
    d.workspace_id,
    d.project_id,
    d.environment_id,
    d.image,
    dt.region,
    d.cpu_millicores,
    d.memory_mib,
    dt.replicas,
    d.desired_state
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
WHERE  dt.region = sqlc.arg(region)
    AND dt.deployment_id = sqlc.arg(deployment_id)
LIMIT 1;
