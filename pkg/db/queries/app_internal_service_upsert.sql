-- name: UpsertAppInternalService :exec
INSERT INTO app_internal_services (
    id,
    workspace_id,
    app_id,
    environment_id,
    region,
    k8s_service_name,
    k8s_namespace,
    port,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(region),
    sqlc.arg(k8s_service_name),
    sqlc.arg(k8s_namespace),
    sqlc.arg(port),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    k8s_service_name = VALUES(k8s_service_name),
    k8s_namespace = VALUES(k8s_namespace),
    port = VALUES(port),
    updated_at = VALUES(updated_at);
