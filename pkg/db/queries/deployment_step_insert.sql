-- name: InsertDeploymentStep :exec
INSERT INTO `deployment_steps` (
    workspace_id,
    project_id,
    environment_id,
    deployment_id,
    step,
    started_at
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(environment_id),
    sqlc.arg(deployment_id),
    sqlc.arg(step),
    sqlc.arg(started_at)
);
