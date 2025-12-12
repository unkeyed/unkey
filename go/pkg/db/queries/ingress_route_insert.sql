-- name: InsertFrontlineRoute :exec
INSERT INTO frontline_routes (
    id,
    project_id,
    deployment_id,
    environment_id,
    hostname,
    sticky,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(project_id),
    sqlc.arg(deployment_id),
    sqlc.arg(environment_id),
    sqlc.arg(hostname),
    sqlc.arg(sticky),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
