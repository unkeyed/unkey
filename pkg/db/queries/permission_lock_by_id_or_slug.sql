-- name: LockPermissionByIdOrSlug :one
(
    SELECT p1.id, p1.project_id, p1.name, p1.slug, p1.description, 0 AS lookup_priority
    FROM permissions p1
    WHERE p1.workspace_id = sqlc.arg(workspace_id) AND p1.id = sqlc.arg(search)
    FOR UPDATE
)
UNION ALL
(
    SELECT p2.id, p2.project_id, p2.name, p2.slug, p2.description, 1 AS lookup_priority
    FROM permissions p2
    WHERE p2.workspace_id = sqlc.arg(workspace_id) AND p2.slug = sqlc.arg(search)
    FOR UPDATE
)
ORDER BY lookup_priority
LIMIT 1;
