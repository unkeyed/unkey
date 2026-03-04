-- name: InsertApp :exec
INSERT INTO apps (
    id,
    workspace_id,
    project_id,
    environment_id,
    name,
    slug,
    current_deployment_id,
    is_rolled_back,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(environment_id),
    sqlc.arg(name),
    sqlc.arg(slug),
    sqlc.arg(current_deployment_id),
    sqlc.arg(is_rolled_back),
    sqlc.arg(delete_protection),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
