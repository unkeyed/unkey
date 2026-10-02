-- name: ListResourceNamesByIDs :many
-- A deleted app or environment does not hide the name of its parent. parent_id
-- is the project id of an app and the app id of an environment. The query also
-- finds an app through its environment and a project through its app. Each
-- branch finds its rows through the id_unique index of its table
SELECT 'project' AS kind, p.id, p.name, '' AS parent_id
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.id IN (sqlc.slice(project_ids))
UNION ALL
SELECT 'project' AS kind, p.id, p.name, '' AS parent_id
FROM apps a
JOIN projects p ON p.id = a.project_id
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND a.id IN (sqlc.slice(project_of_app_ids))
  AND p.workspace_id = sqlc.arg(workspace_id)
UNION ALL
SELECT 'app' AS kind, a.id, a.name, a.project_id AS parent_id
FROM apps a
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND a.id IN (sqlc.slice(app_ids))
UNION ALL
SELECT 'app' AS kind, a.id, a.name, a.project_id AS parent_id
FROM environments e
JOIN apps a ON a.id = e.app_id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.id IN (sqlc.slice(app_of_environment_ids))
  AND a.workspace_id = sqlc.arg(workspace_id)
UNION ALL
SELECT 'environment' AS kind, e.id, e.slug AS name, e.app_id AS parent_id
FROM environments e
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.id IN (sqlc.slice(environment_ids));
