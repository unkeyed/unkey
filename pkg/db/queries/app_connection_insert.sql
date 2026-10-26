-- name: InsertAppConnection :exec
INSERT INTO app_connections (
    id, workspace_id, project_id, app_id, environment_id,
    resource_type, resource_id, name, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
