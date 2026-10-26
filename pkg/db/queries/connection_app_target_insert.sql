-- name: InsertConnectionAppTarget :exec
INSERT INTO connection_app_targets (
    connection_id, selection_mode, target_environment_id, target_deployment_id
) VALUES (?, ?, ?, ?);
