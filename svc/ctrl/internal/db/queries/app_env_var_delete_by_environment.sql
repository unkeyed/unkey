-- name: DeleteAppEnvVarsByEnvironmentId :exec
DELETE FROM app_environment_variables WHERE environment_id = sqlc.arg(environment_id);
