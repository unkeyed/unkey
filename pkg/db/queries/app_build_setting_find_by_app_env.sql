-- name: FindAppBuildSettingByAppEnv :one
SELECT *
FROM `app_build_settings`
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
