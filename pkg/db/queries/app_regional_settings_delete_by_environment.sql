-- name: DeleteAppRegionalSettingsByEnvironmentId :exec
DELETE FROM app_regional_settings WHERE environment_id = sqlc.arg(environment_id);
