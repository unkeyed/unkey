-- name: DeleteAppBuildSettingsByEnvironmentId :exec
DELETE FROM app_build_settings WHERE environment_id = sqlc.arg(environment_id);
