-- name: UpdateConnectionAppTarget :exec
UPDATE connection_app_targets
SET selection_mode = sqlc.arg(selection_mode),
    target_environment_id = sqlc.arg(target_environment_id),
    target_deployment_id = sqlc.arg(target_deployment_id)
WHERE connection_id = sqlc.arg(connection_id);
