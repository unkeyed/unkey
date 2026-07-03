-- name: InsertProject :exec
INSERT INTO projects (
    id,
    workspace_id,
    name,
    slug,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);
