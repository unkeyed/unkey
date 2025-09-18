-- name: InsertDomain :exec
INSERT INTO domains (
    id,
    workspace_id,
    project_id,
    deployment_id,
    domain,
    type,
    sticky,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(deployment_id),
    sqlc.arg(domain),
    sqlc.arg(type),
    sqlc.arg(sticky),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
) ON DUPLICATE KEY UPDATE
    workspace_id = sqlc.arg(workspace_id),
    project_id = sqlc.arg(project_id),
    deployment_id = sqlc.arg(deployment_id),
    type = sqlc.arg(type),
    sticky = sqlc.arg(sticky),
    updated_at = sqlc.arg(updated_at);
