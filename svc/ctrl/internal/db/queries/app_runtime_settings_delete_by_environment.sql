-- name: DeleteAppRuntimeSettingsByEnvironmentId :exec
DELETE FROM app_runtime_settings WHERE environment_id = sqlc.arg(environment_id);
