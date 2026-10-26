-- name: ListSavedAppConnections :many
-- Used by tests to inspect saved deployment connections.
SELECT c.connection_id, c.name, c.resource_id, t.selection_mode,
    t.target_environment_id, t.target_deployment_id
FROM deployment_connections c
INNER JOIN deployment_connection_app_targets t
    ON t.deployment_id = c.deployment_id AND t.connection_id = c.connection_id
WHERE c.workspace_id = sqlc.arg(workspace_id)
    AND c.deployment_id = sqlc.arg(deployment_id)
ORDER BY c.name;
