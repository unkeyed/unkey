-- name: InsertConnectionAppTarget :exec
-- InsertConnectionAppTarget must share a transaction with InsertAppConnection.
INSERT INTO connection_app_targets (
    connection_id, selection_mode, target_environment_id, target_deployment_id
) VALUES (?, ?, ?, ?);
