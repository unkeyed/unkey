-- name: UpsertAppDockerSource :exec
INSERT INTO app_docker_sources (
    workspace_id,
    app_id,
    image,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(image),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    image = VALUES(image),
    updated_at = VALUES(updated_at);
