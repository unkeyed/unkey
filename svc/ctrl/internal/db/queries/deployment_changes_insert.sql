-- name: InsertDeploymentChange :exec
INSERT INTO `deployment_changes` (
    resource_type,
    resource_id,
    region_id,
    created_at
) VALUES (
    sqlc.arg(resource_type),
    sqlc.arg(resource_id),
    sqlc.arg(region_id),
    sqlc.arg(created_at)
);
