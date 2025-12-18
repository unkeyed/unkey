-- name: FindDeploymentTopologyByIDAndRegion :one
SELECT
    d.id,
    d.k8s_name,
    w.k8s_namespace,
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
INNER JOIN `workspaces` w ON d.workspace_id = w.id
WHERE  dt.region = sqlc.arg(region)
    AND dt.deployment_id = sqlc.arg(deployment_id)
LIMIT 1;
