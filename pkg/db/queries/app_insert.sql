-- name: InsertApp :exec
INSERT INTO apps (
    id,
    workspace_id,
    project_id,
    name,
    slug,
    live_deployment_id,
    depot_project_id,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(name),
    sqlc.arg(slug),
    sqlc.arg(live_deployment_id),
    sqlc.arg(depot_project_id),
    sqlc.arg(delete_protection),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
