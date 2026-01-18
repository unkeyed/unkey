-- name: InsertStateChange :execlastid
INSERT INTO `state_changes` (
    resource_type,
    state,
    cluster_id,
    created_at
) VALUES (
    sqlc.arg(resource_type),
    sqlc.arg(state),
    sqlc.arg(cluster_id),
    sqlc.arg(created_at)
);
