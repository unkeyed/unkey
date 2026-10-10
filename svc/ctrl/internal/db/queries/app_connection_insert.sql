-- name: InsertAppConnection :exec
INSERT INTO app_connections (
    id,
    workspace_id,
    project_id,
    app_id,
    environment_id,
    resource_type,
    resource_id,
    name,
    created_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(resource_type),
    sqlc.arg(resource_id),
    sqlc.arg(name),
    sqlc.arg(created_at)
);
