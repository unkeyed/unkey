-- name: FindAppRuntimeSettingsByAppAndEnv :one
SELECT
  app_runtime_settings.pk,
  app_runtime_settings.workspace_id,
  app_runtime_settings.app_id,
  app_runtime_settings.environment_id,
  app_runtime_settings.port,
  app_runtime_settings.cpu_millicores,
  app_runtime_settings.memory_mib,
  app_runtime_settings.storage_mib,
  app_runtime_settings.command,
  app_runtime_settings.healthcheck,
  app_runtime_settings.shutdown_signal,
  app_runtime_settings.upstream_protocol,
  app_runtime_settings.sentinel_config,
  app_runtime_settings.openapi_spec_path,
  app_runtime_settings.created_at,
  app_runtime_settings.updated_at
FROM app_runtime_settings
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
