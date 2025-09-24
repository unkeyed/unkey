-- name: InsertDomain :exec
INSERT INTO domains (
    id,
    workspace_id,
    project_id,
    environment_id,
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
    sqlc.arg(environment_id),
    sqlc.arg(deployment_id),
    sqlc.arg(domain),
    sqlc.arg(type),
    sqlc.arg(sticky),
    sqlc.arg(created_at),
    null
);
