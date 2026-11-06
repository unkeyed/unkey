-- name: FindAppConnectionPinnedDeployment :one
SELECT d.id, d.environment_id, d.status, d.git_branch, d.image_resolved
FROM app_connections c
INNER JOIN connection_app_targets t ON t.connection_id = c.id
INNER JOIN deployments d ON d.id = t.target_deployment_id
WHERE c.workspace_id = sqlc.arg(workspace_id) AND c.app_id = sqlc.arg(app_id)
    AND c.environment_id = sqlc.arg(environment_id)
    AND c.resource_type = 'app' AND c.resource_id = sqlc.arg(target_app_id)
    AND t.selection_mode = 'deployment'
    AND d.workspace_id = sqlc.arg(workspace_id) AND d.app_id = sqlc.arg(target_app_id);
