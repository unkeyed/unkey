-- name: InsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    app_id,
    slug,
    description,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);
