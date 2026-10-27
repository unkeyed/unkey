-- name: ListDeploymentConnectionSummaries :many
SELECT b.connection_id AS id, a.name AS target_app_name, t.selection_mode AS target_type,
    t.target_environment_id, e.slug AS target_environment_slug, t.target_deployment_id
FROM deployment_connections b
INNER JOIN deployments d ON d.id = b.deployment_id
    AND d.workspace_id = b.workspace_id AND d.project_id = b.project_id
    AND d.app_id = b.app_id AND d.environment_id = b.environment_id
INNER JOIN deployment_connection_app_targets t
    ON t.deployment_id = b.deployment_id AND t.connection_id = b.connection_id
LEFT JOIN apps a ON a.id = b.resource_id
    AND a.workspace_id = sqlc.arg(workspace_id) AND a.project_id = sqlc.arg(project_id)
LEFT JOIN environments e ON e.id = t.target_environment_id AND e.app_id = a.id
    AND e.workspace_id = sqlc.arg(workspace_id) AND e.project_id = sqlc.arg(project_id)
WHERE b.deployment_id = sqlc.arg(deployment_id)
    AND b.workspace_id = sqlc.arg(workspace_id) AND b.project_id = sqlc.arg(project_id)
    AND b.resource_type = 'app'
ORDER BY b.name ASC, b.connection_id ASC
LIMIT 6;
