-- name: ListDesiredDeploymentTopology :many
SELECT
    d.id as deployment_id,
    d.workspace_id,
    d.project_id,
    d.environment_id,
    d.image,
    dt.region,
    d.cpu_millicores,
    d.memory_mib,
    dt.replicas
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
WHERE (sqlc.arg(region) = '' OR dt.region = sqlc.arg(region))
    AND d.desired_state = sqlc.arg(desired_state)
    AND dt.deployment_id > sqlc.arg(pagination_cursor)
ORDER BY dt.deployment_id ASC
LIMIT ?;
