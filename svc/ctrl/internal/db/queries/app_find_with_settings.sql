-- name: FindAppWithSettings :one
SELECT
    a.id AS app_id,
    a.project_id AS app_project_id,
    a.source_type AS app_source_type,
    a.default_branch AS app_default_branch,
    a.current_deployment_id AS app_current_deployment_id,
    abs.dockerfile AS build_settings_dockerfile,
    abs.docker_context AS build_settings_docker_context,
    abs.build_command AS build_settings_build_command,
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
INNER JOIN app_build_settings abs ON abs.app_id = a.id AND abs.environment_id = sqlc.arg(environment_id)
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = sqlc.arg(environment_id)
WHERE a.id = sqlc.arg(id);
