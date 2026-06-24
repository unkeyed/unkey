-- name: FindProjectByIdOrSlug :one
SELECT
    p.id,
    p.workspace_id,
    p.name,
    p.slug,
    p.delete_protection,
    p.created_at,
    p.updated_at
FROM projects p
JOIN (
    SELECT p1.id
    FROM projects p1
    WHERE p1.id = sqlc.arg(project) AND p1.workspace_id = sqlc.arg(workspace_id)
    UNION ALL
    SELECT p2.id
    FROM projects p2
    WHERE p2.slug = sqlc.arg(project) AND p2.workspace_id = sqlc.arg(workspace_id)
) AS project_lookup ON p.id = project_lookup.id
LIMIT 1;
