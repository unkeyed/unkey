-- name: ResolveDeploymentScope :one
-- Resolves the optional project/app/environment filters (each an id or slug) to
-- concrete ids for the listDeployments endpoint, in one query. project is
-- required; app and environment are optional and resolved via LEFT JOIN, so a
-- level that is absent or does not match comes back NULL. The caller reads a
-- NULL id for a level it did request as "not found". Hierarchy is enforced by
-- the join keys: an app must belong to the resolved project, an environment to
-- the resolved app.
-- The project is resolved through a UNION of an id seek and a slug seek (each
-- hits an index) rather than `id = ? OR slug = ?`, which would scan every
-- project in the workspace. app/environment keep the OR-guard because the join
-- first seeks them by their parent id (project_id, app_id) down to a handful of
-- rows, so the residual id-or-slug filter is cheap.
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
