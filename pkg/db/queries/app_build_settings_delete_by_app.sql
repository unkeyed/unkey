-- name: DeleteAppBuildSettingsByAppId :exec
DELETE FROM app_build_settings WHERE app_id = sqlc.arg(app_id);
