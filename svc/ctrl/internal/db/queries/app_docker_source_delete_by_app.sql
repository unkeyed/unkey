-- name: DeleteAppDockerSourceByAppId :exec
DELETE FROM app_docker_sources WHERE app_id = sqlc.arg(app_id);
