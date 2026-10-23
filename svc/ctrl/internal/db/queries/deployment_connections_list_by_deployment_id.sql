-- name: ListDeploymentConnectionsByDeploymentId :many
-- ListDeploymentConnectionsByDeploymentId returns the connections a deployment
-- saved with their app targets. Only tests use it.
SELECT
    b.connection_id,
    b.name,
    b.resource_id,
    t.selection_mode,
    t.target_environment_id,
    t.target_deployment_id
FROM deployment_connections b
LEFT JOIN deployment_connection_app_targets t
    ON t.deployment_id = b.deployment_id AND t.connection_id = b.connection_id
WHERE b.deployment_id = sqlc.arg(deployment_id)
ORDER BY b.name;
