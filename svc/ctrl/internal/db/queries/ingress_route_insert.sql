-- name: InsertFrontlineRoute :exec
-- Keeps an existing row for the domain instead of failing the unique
-- fully-qualified-domain-name key, so two deployments of the same domain
-- converge on the row that was created first.
INSERT INTO frontline_routes (
    id,
    project_id,
    app_id,
    deployment_id,
    environment_id,
    fully_qualified_domain_name,
    sticky,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(deployment_id),
    sqlc.arg(environment_id),
    sqlc.arg(fully_qualified_domain_name),
    sqlc.arg(sticky),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    id = id;
