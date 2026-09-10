-- name: FindEnvironmentByIdentifiers :one
SELECT environments.pk, environments.id, environments.workspace_id, environments.project_id, environments.app_id, environments.slug, environments.description, environments.kind, environments.delete_protection, environments.created_at, environments.updated_at
FROM environments
JOIN apps a ON environments.app_id = a.id AND environments.workspace_id = a.workspace_id
JOIN projects p ON a.project_id = p.id AND a.workspace_id = p.workspace_id
WHERE environments.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
  AND (environments.id = sqlc.arg(environment) OR environments.slug = sqlc.arg(environment))
LIMIT 1;
