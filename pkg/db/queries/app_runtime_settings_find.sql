-- name: FindAppRuntimeSettingsByAppAndEnv :one
SELECT sqlc.embed(app_runtime_settings)
FROM app_runtime_settings
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
