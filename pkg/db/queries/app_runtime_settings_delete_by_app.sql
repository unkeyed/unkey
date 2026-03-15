-- name: DeleteAppRuntimeSettingsByAppId :exec
DELETE FROM app_runtime_settings WHERE app_id = sqlc.arg(app_id);
