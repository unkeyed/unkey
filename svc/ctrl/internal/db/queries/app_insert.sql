-- name: InsertApp :exec
INSERT INTO apps (
    id,
    workspace_id,
    project_id,
    name,
    slug,
    source_type,
    default_branch,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(name),
    sqlc.arg(slug),
    sqlc.arg(source_type),
    sqlc.arg(default_branch),
    sqlc.arg(delete_protection),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
