-- name: InsertPermission :exec
INSERT INTO `permissions` (
    id,
    workspace_id,
    name,
    description,
    created_at_m
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(name),
    sqlc.arg(description),
    sqlc.arg(created_at)
);
