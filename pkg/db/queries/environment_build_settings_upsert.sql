-- name: UpsertEnvironmentBuildSettings :exec
INSERT INTO environment_build_settings (
    id,
    workspace_id,
    environment_id,
    dockerfile,
    docker_context,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    dockerfile = VALUES(dockerfile),
    docker_context = VALUES(docker_context),
    updated_at = VALUES(updated_at);
