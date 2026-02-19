-- name: UpsertAppRuntimeSettings :exec
INSERT INTO app_runtime_settings (
    workspace_id,
    app_id,
    environment_id,
    port,
    cpu_millicores,
    memory_mib,
    command,
    healthcheck,
    region_config,
    shutdown_signal,
    sentinel_config,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(port),
    sqlc.arg(cpu_millicores),
    sqlc.arg(memory_mib),
    sqlc.arg(command),
    sqlc.arg(healthcheck),
    sqlc.arg(region_config),
    sqlc.arg(shutdown_signal),
    sqlc.arg(sentinel_config),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    port = VALUES(port),
    cpu_millicores = VALUES(cpu_millicores),
    memory_mib = VALUES(memory_mib),
    command = VALUES(command),
    healthcheck = VALUES(healthcheck),
    region_config = VALUES(region_config),
    shutdown_signal = VALUES(shutdown_signal),
    sentinel_config = VALUES(sentinel_config),
    updated_at = VALUES(updated_at);
