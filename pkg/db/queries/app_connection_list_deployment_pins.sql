-- name: ListCurrentDeploymentConnectionPins :many
SELECT b.id AS connection_id, b.name AS connection_name,
    b.app_id AS caller_app_id, a.name AS caller_app_name
FROM app_connections b
INNER JOIN connection_app_targets t ON t.connection_id = b.id
INNER JOIN apps a ON a.id = b.app_id
    AND a.workspace_id = sqlc.arg(workspace_id) AND a.project_id = sqlc.arg(project_id)
WHERE b.workspace_id = sqlc.arg(workspace_id) AND b.project_id = sqlc.arg(project_id)
    AND b.resource_type = 'app' AND b.resource_id = sqlc.arg(target_app_id)
    AND t.selection_mode = 'deployment' AND t.target_deployment_id = sqlc.arg(deployment_id)
ORDER BY b.name ASC
LIMIT 11;
