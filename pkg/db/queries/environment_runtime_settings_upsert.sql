-- name: UpsertEnvironmentRuntimeSettings :exec
INSERT INTO environment_runtime_settings (
    id,
    workspace_id,
    environment_id,
    port,
    cpu_millicores,
    memory_mib,
    command,
    healthcheck,
    region_config,
    restart_policy,
    shutdown_signal,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    port = VALUES(port),
    cpu_millicores = VALUES(cpu_millicores),
    memory_mib = VALUES(memory_mib),
    command = VALUES(command),
    healthcheck = VALUES(healthcheck),
    region_config = VALUES(region_config),
    restart_policy = VALUES(restart_policy),
    shutdown_signal = VALUES(shutdown_signal),
    updated_at = VALUES(updated_at);
