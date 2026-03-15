-- name: DeleteAppEnvVarsByAppId :exec
DELETE FROM app_environment_variables WHERE app_id = sqlc.arg(app_id);
