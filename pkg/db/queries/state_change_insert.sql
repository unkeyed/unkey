-- name: InsertStateChange :execlastid
INSERT INTO `state_changes` (
    resource_type,
    resource_id,
    op,
    region,
    created_at
) VALUES (
    sqlc.arg(resource_type),
    sqlc.arg(resource_id),
    sqlc.arg(op),
    sqlc.arg(region),
    sqlc.arg(created_at)
);
