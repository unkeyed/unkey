-- name: InsertDeploymentTopology :exec
INSERT INTO `deployment_topology` (
    workspace_id,
    deployment_id,
    region,
    region_id,
    desired_replicas,
    desired_status,
    version,
    created_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(deployment_id),
    sqlc.arg(region),
    sqlc.arg(region_id),
    sqlc.arg(desired_replicas),
    sqlc.arg(desired_status),
    sqlc.arg(version),
    sqlc.arg(created_at)
);
