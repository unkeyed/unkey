-- name: ListRetainedAppConnections :many
SELECT b.connection_id AS id, a.name AS target_app_name,
    COUNT(DISTINCT b.deployment_id) AS deployments
FROM deployment_connections b
INNER JOIN apps a ON a.id = b.resource_id
INNER JOIN deployments d ON d.id = b.deployment_id
    AND d.workspace_id = sqlc.arg(workspace_id)
    AND d.app_id = sqlc.arg(app_id)
    AND d.environment_id = sqlc.arg(environment_id)
LEFT JOIN app_connections current_rule ON current_rule.id = b.connection_id
WHERE b.workspace_id = sqlc.arg(workspace_id)
    AND b.project_id = sqlc.arg(project_id)
    AND b.app_id = sqlc.arg(app_id)
    AND b.environment_id = sqlc.arg(environment_id)
    AND b.resource_type = 'app'
    AND a.workspace_id = sqlc.arg(workspace_id)
    AND a.project_id = sqlc.arg(project_id)
    AND current_rule.id IS NULL
GROUP BY b.connection_id, a.name
ORDER BY b.connection_id ASC
LIMIT 500;
