-- name: UpsertEnvironmentBuildSettings :exec
INSERT INTO environment_build_settings (
    id,
    workspace_id,
    environment_id,
    dockerfile,
    docker_context,
    build_cpu_millicores,
    build_memory_mib,
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    dockerfile = VALUES(dockerfile),
    docker_context = VALUES(docker_context),
    build_cpu_millicores = VALUES(build_cpu_millicores),
    build_memory_mib = VALUES(build_memory_mib);
