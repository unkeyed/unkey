-- name: InsertDeploymentTopology :exec
INSERT INTO `deployment_topology` (
    workspace_id,
    deployment_id,
    region_id,
    autoscaling_replicas_min,
    autoscaling_replicas_max,
    autoscaling_threshold_cpu,
    autoscaling_threshold_memory,
    desired_status,
    version,
    created_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(deployment_id),
    sqlc.arg(region_id),
    sqlc.arg(autoscaling_replicas_min),
    sqlc.arg(autoscaling_replicas_max),
    sqlc.arg(autoscaling_threshold_cpu),
    sqlc.arg(autoscaling_threshold_memory),
    sqlc.arg(desired_status),
    sqlc.arg(version),
    sqlc.arg(created_at)
);
