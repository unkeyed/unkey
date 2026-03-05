-- name: InsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    app_id,
    slug,
    description,
    current_deployment_id,
    is_rolled_back,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);
