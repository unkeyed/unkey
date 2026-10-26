-- name: ListSnapshotDeploymentConnectionPins :many
SELECT b.connection_id, b.name AS connection_name, b.app_id AS caller_app_id,
    a.name AS caller_app_name, d.id AS caller_deployment_id, d.status AS caller_status
FROM deployment_connections b
INNER JOIN deployment_connection_app_targets t
    ON t.deployment_id = b.deployment_id AND t.connection_id = b.connection_id
INNER JOIN deployments d ON d.id = b.deployment_id
    AND d.workspace_id = b.workspace_id AND d.project_id = b.project_id
    AND d.app_id = b.app_id AND d.environment_id = b.environment_id
INNER JOIN environments e ON e.id = b.environment_id AND e.app_id = b.app_id
    AND e.workspace_id = sqlc.arg(workspace_id)
INNER JOIN apps a ON a.id = d.app_id
    AND a.workspace_id = sqlc.arg(workspace_id) AND a.project_id = sqlc.arg(project_id)
WHERE b.workspace_id = sqlc.arg(workspace_id) AND b.project_id = sqlc.arg(project_id)
    AND b.resource_type = 'app' AND b.resource_id = sqlc.arg(target_app_id)
    AND t.selection_mode = 'deployment' AND t.target_deployment_id = sqlc.arg(deployment_id)
    AND d.desired_state = 'running'
    AND d.status IN ('pending', 'starting', 'building', 'deploying', 'network', 'finalizing', 'ready')
    AND JSON_CONTAINS(d.capabilities, 'true', '$.private_networking')
ORDER BY b.name ASC, d.created_at ASC
LIMIT 11;
