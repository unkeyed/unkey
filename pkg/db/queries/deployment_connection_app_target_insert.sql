-- name: InsertDeploymentConnectionAppTarget :exec
-- Used by tests to seed saved deployment targets.
INSERT INTO deployment_connection_app_targets (
    deployment_id, connection_id, selection_mode, target_environment_id, target_deployment_id
) VALUES (?, ?, ?, ?, ?);
