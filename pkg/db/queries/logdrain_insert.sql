-- name: InsertLogdrain :exec
-- InsertLogdrain creates a running log drain with no lease. Intended for
-- tests: the dashboard creates log drains in production.
INSERT INTO logdrains (
    id,
    workspace_id,
    name,
    stream,
    config,
    lease_id,
    fencing_token,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(name),
    sqlc.arg(stream),
    sqlc.arg(config),
    '',
    '',
    sqlc.arg(created_at)
);
