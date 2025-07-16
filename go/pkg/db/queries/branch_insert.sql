-- name: InsertBranch :exec
INSERT INTO branches (
    id,
    workspace_id,
    project_id,
    name,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?
);