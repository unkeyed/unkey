-- name: InsertDeploymentStep :exec
INSERT INTO deployment_steps (
    workspace_id,
    project_id,
    deployment_id,
    status,
    message,
    created_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(deployment_id),
    sqlc.arg(status),
    sqlc.arg(message),
    sqlc.arg(created_at)
)
ON DUPLICATE KEY UPDATE
    message = VALUES(message),
    created_at = VALUES(created_at);
