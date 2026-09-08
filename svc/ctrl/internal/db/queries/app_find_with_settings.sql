-- name: FindAppWithRuntimeSettings :one
SELECT
    a.id AS app_id,
    a.project_id AS app_project_id,
    a.source_type AS app_source_type,
    a.current_deployment_id AS app_current_deployment_id,
    ars.port AS runtime_settings_port,
    ars.cpu_millicores AS runtime_settings_cpu_millicores,
    ars.memory_mib AS runtime_settings_memory_mib,
    ars.storage_mib AS runtime_settings_storage_mib,
    ars.command AS runtime_settings_command,
    ars.healthcheck AS runtime_settings_healthcheck,
    ars.shutdown_signal AS runtime_settings_shutdown_signal,
    ars.upstream_protocol AS runtime_settings_upstream_protocol,
    ars.sentinel_config AS runtime_settings_sentinel_config
FROM apps a
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = sqlc.arg(environment_id)
WHERE a.id = sqlc.arg(id);
