-- name: FindAppByProjectAndIdOrSlug :one
SELECT sqlc.embed(a)
FROM apps a
JOIN projects p ON p.id = a.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app));
