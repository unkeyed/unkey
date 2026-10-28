-- name: ListAppConnections :many
SELECT c.id, c.app_id, c.environment_id, c.resource_id AS target_app_id,
    a.name AS target_app_name, a.slug AS target_app_slug, c.name,
    t.selection_mode, t.target_environment_id, t.target_deployment_id
FROM app_connections c
LEFT JOIN connection_app_targets t ON t.connection_id = c.id
INNER JOIN apps a ON a.id = c.resource_id
    AND a.workspace_id = c.workspace_id AND a.project_id = c.project_id
WHERE c.workspace_id = sqlc.arg(workspace_id)
    AND c.project_id = sqlc.arg(project_id)
    AND c.app_id = sqlc.arg(app_id)
    AND c.environment_id = sqlc.arg(environment_id)
    AND c.resource_type = 'app'
    AND c.resource_id <> c.app_id
    AND c.id >= sqlc.arg(id_cursor)
ORDER BY c.id
LIMIT ?;
