-- name: FindEnvironmentByAppAndIdOrSlug :one
SELECT sqlc.embed(e)
FROM environments e
JOIN apps a ON a.id = e.app_id
JOIN projects p ON p.id = a.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
  AND (e.id = sqlc.arg(environment) OR e.slug = sqlc.arg(environment));
