-- name: FindAppRuntimeSettingsByAppAndEnv :one
SELECT app_runtime_settings.openapi_spec_path
FROM app_runtime_settings
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
