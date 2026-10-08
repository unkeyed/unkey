-- name: FindPermissionByIdOrSlug :one
-- FindPermissionByIdOrSlug resolves a permission within a workspace so the
-- caller can authorize access against the permission's actual project.
(
    SELECT p1.pk, p1.id, p1.workspace_id, p1.project_id, p1.name, p1.slug, p1.description, p1.created_at_m, p1.updated_at_m, 0 AS lookup_priority
    FROM permissions p1
    WHERE p1.workspace_id = sqlc.arg(workspace_id) AND p1.id = sqlc.arg(search)
)
UNION ALL
(
    SELECT p2.pk, p2.id, p2.workspace_id, p2.project_id, p2.name, p2.slug, p2.description, p2.created_at_m, p2.updated_at_m, 1 AS lookup_priority
    FROM permissions p2
    WHERE p2.workspace_id = sqlc.arg(workspace_id) AND p2.slug = sqlc.arg(search)
)
ORDER BY lookup_priority
LIMIT 1;
