-- name: UpsertAppBuildSettings :exec
INSERT INTO app_build_settings (
    workspace_id,
    app_id,
    environment_id,
    dockerfile,
    docker_context,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(dockerfile),
    sqlc.arg(docker_context),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    dockerfile = VALUES(dockerfile),
    docker_context = VALUES(docker_context),
    updated_at = VALUES(updated_at);
