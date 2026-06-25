-- name: DeleteAppRegionalSettingByAppEnvRegion :exec
DELETE FROM app_regional_settings
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND region_id = sqlc.arg(region_id);
