-- name: InsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    slug,
    description,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);