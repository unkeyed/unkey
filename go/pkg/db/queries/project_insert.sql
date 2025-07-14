-- name: InsertProject :exec
INSERT INTO projects (
    id,
    workspace_id,
    partition_id,
    name,
    slug,
    git_repository_url,
    default_branch,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);