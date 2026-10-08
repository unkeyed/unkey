-- name: DeleteDeploymentConnectionsByWorkspaceIds :exec
-- DeleteDeploymentConnectionsByWorkspaceIds deletes the connections saved by
-- the workspaces' deployments. It reads deployments by workspace_idx, so run it
-- before those deployments are deleted.
DELETE c, ct
FROM deployments d
STRAIGHT_JOIN deployment_connections c ON c.deployment_id = d.id
LEFT JOIN deployment_connection_app_targets ct
    ON ct.deployment_id = c.deployment_id AND ct.connection_id = c.connection_id
WHERE d.workspace_id IN (sqlc.slice('ids'));
