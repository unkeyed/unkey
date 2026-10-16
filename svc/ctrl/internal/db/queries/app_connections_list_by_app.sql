-- name: ListAppConnectionsByApp :many
-- Keep incomplete app connections so deployment admission rejects missing target settings.
SELECT b.id, b.name, b.resource_type, b.resource_id, t.selection_mode,
    t.target_environment_id, t.target_deployment_id
FROM app_connections b
LEFT JOIN connection_app_targets t ON t.connection_id = b.id
INNER JOIN apps a ON a.id = b.app_id
WHERE b.workspace_id = sqlc.arg(workspace_id)
    AND b.project_id = sqlc.arg(project_id)
    AND b.app_id = sqlc.arg(app_id)
    AND b.environment_id = sqlc.arg(environment_id)
    AND b.resource_type = 'app'
    AND b.resource_id <> b.app_id
    AND b.name <> a.slug COLLATE utf8mb4_0900_as_cs
ORDER BY b.pk;
