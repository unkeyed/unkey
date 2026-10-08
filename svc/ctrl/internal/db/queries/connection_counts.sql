-- name: CountDeploymentConnectionsByEnvironmentId :one
-- CountDeploymentConnectionsByEnvironmentId verifies deletion cascades in tests.
SELECT COUNT(*) FROM deployment_connections WHERE environment_id = sqlc.arg(environment_id);

-- name: CountDeploymentConnectionsByWorkspaceId :one
-- CountDeploymentConnectionsByWorkspaceId verifies deletion cascades in tests.
SELECT COUNT(*) FROM deployment_connections WHERE workspace_id = sqlc.arg(workspace_id);

-- name: CountOrphanConnectionAppTargets :one
-- CountOrphanConnectionAppTargets verifies deletion cascades in tests.
SELECT COUNT(*) FROM connection_app_targets t
LEFT JOIN app_connections b ON b.id = t.connection_id WHERE b.id IS NULL;

-- name: CountOrphanDeploymentConnectionAppTargets :one
-- CountOrphanDeploymentConnectionAppTargets verifies deletion cascades in tests.
SELECT COUNT(*) FROM deployment_connection_app_targets t
LEFT JOIN deployment_connections b ON b.deployment_id = t.deployment_id AND b.connection_id = t.connection_id
WHERE b.pk IS NULL;

-- name: CountDeploymentConnectionAppTargetsByConnectionId :one
-- CountDeploymentConnectionAppTargetsByConnectionId verifies deletion isolation in tests.
SELECT COUNT(*) FROM deployment_connection_app_targets WHERE connection_id = sqlc.arg(connection_id);

-- name: CountAppConnectionsByWorkspaceId :one
-- CountAppConnectionsByWorkspaceId verifies deletion cascades in tests.
SELECT COUNT(*) FROM app_connections WHERE workspace_id = sqlc.arg(workspace_id);

-- name: CountAppConnectionsByResource :one
-- CountAppConnectionsByResource verifies resource-type isolation in tests.
SELECT COUNT(*) FROM app_connections
WHERE resource_id = sqlc.arg(resource_id) AND (sqlc.arg(resource_type) = '' OR resource_type = sqlc.arg(resource_type));

-- name: CountDeploymentConnectionsByDeploymentId :one
-- CountDeploymentConnectionsByDeploymentId verifies environment deletion in tests.
SELECT COUNT(*) FROM deployment_connections WHERE deployment_id = sqlc.arg(deployment_id);

-- name: CountDeploymentConnectionAppTargetsByDeploymentId :one
-- CountDeploymentConnectionAppTargetsByDeploymentId verifies environment deletion in tests.
SELECT COUNT(*) FROM deployment_connection_app_targets WHERE deployment_id = sqlc.arg(deployment_id);
