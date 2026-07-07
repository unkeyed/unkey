-- name: ResolveDeploymentScope :one
-- Resolves a project (required) + optional app/environment, each an id or slug, to
-- their ids in one query. app/environment LEFT JOIN on the parent id, so a value that
-- doesn't match yields NULL for that level (caller reads NULL as not-found).
-- Project uses UNION ALL of two index seeks instead of `id = ? OR slug = ?`, which
-- can't use both indexes and would scan the workspace. app/environment keep the OR
-- since the parent-id join already narrows them to a few rows.
SELECT
    p.id AS project_id,
    a.id AS app_id,
    e.id AS environment_id
FROM (
    SELECT p1.id, p1.workspace_id
    FROM projects p1
    WHERE p1.workspace_id = sqlc.arg(workspace_id) AND p1.id = sqlc.arg(project)
    UNION ALL
    SELECT p2.id, p2.workspace_id
    FROM projects p2
    WHERE p2.workspace_id = sqlc.arg(workspace_id) AND p2.slug = sqlc.arg(project)
    LIMIT 1
) p
LEFT JOIN apps a
    ON a.project_id = p.id
    AND a.workspace_id = p.workspace_id
    AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
LEFT JOIN environments e
    ON e.app_id = a.id
    AND e.workspace_id = a.workspace_id
    AND (e.id = sqlc.arg(environment) OR e.slug = sqlc.arg(environment))
LIMIT 1;
