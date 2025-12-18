-- name: InsertSentinel :exec
INSERT INTO sentinels (
    id,
    workspace_id,
    environment_id,
    project_id,
    k8s_address,
    k8s_name,
    region,
    image,
    health,
    desired_replicas,
    replicas,
    cpu_millicores,
    memory_mib,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(environment_id),
    sqlc.arg(project_id),
    sqlc.arg(k8s_address),
    sqlc.arg(k8s_name),
    sqlc.arg(region),
    sqlc.arg(image),
    sqlc.arg(health),
    sqlc.arg(desired_replicas),
    sqlc.arg(replicas),
    sqlc.arg(cpu_millicores),
    sqlc.arg(memory_mib),
    sqlc.arg(created_at)
);
