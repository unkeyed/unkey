-- name: InsertCiliumNetworkPolicy :exec
INSERT INTO cilium_network_policies (
    id,
    workspace_id,
    project_id,
    environment_id,
    k8s_name,
    region,
    policy,
    version,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(environment_id),
    sqlc.arg(k8s_name),
    sqlc.arg(region),
    sqlc.arg(policy),
    sqlc.arg(version),
    sqlc.arg(created_at)
);
