-- name: DeleteAppRegionalSettingsByAppId :exec
DELETE FROM app_regional_settings WHERE app_id = sqlc.arg(app_id);
