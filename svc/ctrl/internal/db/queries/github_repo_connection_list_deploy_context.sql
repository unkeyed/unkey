-- name: ListRepoConnectionDeployContexts :many
SELECT
    gc.installation_id AS connection_installation_id,
    gc.repository_full_name AS connection_repository_full_name,
    p.id AS project_id,
    p.workspace_id AS project_workspace_id,
    e.id AS environment_id,
    e.slug AS environment_slug,
    a.id AS app_id,
    abs.auto_deploy AS build_settings_auto_deploy,
    abs.watch_paths AS build_settings_watch_paths,
    abs.docker_context AS build_settings_docker_context,
    abs.dockerfile AS build_settings_dockerfile,
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
FROM github_repo_connections gc
INNER JOIN apps a ON a.id = gc.app_id
INNER JOIN projects p ON p.id = gc.project_id
INNER JOIN environments e ON e.app_id = a.id
  AND e.slug = CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(a.default_branch, ''), 'main')
    THEN 'production'
    ELSE 'preview'
  END
INNER JOIN app_build_settings abs ON abs.app_id = a.id AND abs.environment_id = e.id
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = e.id
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id);
