-- name: InsertCiliumNetworkPolicy :exec
INSERT INTO cilium_network_policies (
    id,
    workspace_id,
    project_id,
    app_id,
    environment_id,
    deployment_id,
    k8s_name,
    k8s_namespace,
    region_id,
    policy,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(deployment_id),
    sqlc.arg(k8s_name),
    sqlc.arg(k8s_namespace),
    sqlc.arg(region_id),
    sqlc.arg(policy),
    sqlc.arg(created_at)
);
