-- name: InsertDeploymentConnectionAppTarget :exec
-- InsertDeploymentConnectionAppTarget must share a transaction with InsertDeploymentConnection.
INSERT INTO deployment_connection_app_targets (
    deployment_id, connection_id, selection_mode, target_environment_id, target_deployment_id
) VALUES (?, ?, ?, ?, ?);
