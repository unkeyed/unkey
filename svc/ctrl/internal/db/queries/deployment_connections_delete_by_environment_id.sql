-- name: DeleteDeploymentConnectionsByEnvironmentId :exec
-- DeleteDeploymentConnectionsByEnvironmentId deletes the connections saved by
-- the environment's deployments. app_id is the environment's app; filtering on
-- it lets the delete use deployment_connections_app_env_idx.
DELETE c, ct
FROM deployment_connections c
LEFT JOIN deployment_connection_app_targets ct
    ON ct.deployment_id = c.deployment_id AND ct.connection_id = c.connection_id
WHERE c.app_id = sqlc.arg(app_id)
    AND c.environment_id = sqlc.arg(environment_id);
