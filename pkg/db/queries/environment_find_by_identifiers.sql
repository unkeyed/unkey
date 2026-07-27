-- name: FindEnvironmentByIdentifiers :one
SELECT e.pk, e.id, e.workspace_id, e.project_id, e.app_id, e.slug, e.description, e.delete_protection, e.created_at, e.updated_at
FROM environments e
JOIN apps a ON e.app_id = a.id AND e.workspace_id = a.workspace_id
JOIN projects p ON a.project_id = p.id AND a.workspace_id = p.workspace_id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
  AND (e.id = sqlc.arg(environment) OR e.slug = sqlc.arg(environment))
LIMIT 1;
