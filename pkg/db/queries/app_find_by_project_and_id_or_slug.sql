-- name: FindAppByProjectAndIdOrSlug :one
SELECT a.pk, a.id, a.workspace_id, a.project_id, a.name, a.slug, a.default_branch, a.current_deployment_id, a.is_rolled_back, a.delete_protection, a.created_at, a.updated_at
FROM apps a
JOIN projects p ON a.project_id = p.id AND a.workspace_id = p.workspace_id
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
LIMIT 1;
