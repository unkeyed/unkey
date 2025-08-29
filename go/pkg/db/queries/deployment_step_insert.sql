-- name: InsertDeploymentStep :exec
INSERT INTO deployment_steps (
    workspace_id,
    project_id,
    deployment_id,
    status,
    message,
    created_at
) VALUES (
    ?, ?, ?, ?, ?, ?
)
ON DUPLICATE KEY UPDATE
    message = VALUES(message),
    created_at = VALUES(created_at);
