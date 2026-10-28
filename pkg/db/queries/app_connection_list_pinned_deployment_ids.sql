-- name: ListAppConnectionPinnedDeploymentIDs :many
SELECT t.target_deployment_id
FROM app_connections b
INNER JOIN connection_app_targets t ON t.connection_id = b.id
INNER JOIN apps a ON a.id = b.resource_id
WHERE b.workspace_id = sqlc.arg(workspace_id) AND b.project_id = sqlc.arg(project_id)
    AND b.app_id = sqlc.arg(app_id) AND b.environment_id = sqlc.arg(environment_id)
    AND b.resource_type = 'app' AND b.resource_id <> b.app_id
    AND a.workspace_id = sqlc.arg(workspace_id) AND a.project_id = sqlc.arg(project_id)
    AND t.selection_mode = 'deployment' AND t.target_deployment_id IS NOT NULL
ORDER BY b.name ASC
LIMIT 500;
