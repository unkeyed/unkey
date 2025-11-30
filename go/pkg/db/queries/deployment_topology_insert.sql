-- name: InsertDeploymentTopology :exec
INSERT INTO `deployment_topology` (
    workspace_id,
    deployment_id,
    region,
    replicas,
    status,
    created_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(deployment_id),
    sqlc.arg(region),
    sqlc.arg(replicas),
    sqlc.arg(status),
    sqlc.arg(created_at)
);
