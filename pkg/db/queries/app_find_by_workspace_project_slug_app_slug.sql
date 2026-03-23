-- name: FindAppByWorkspaceAndSlugs :one
SELECT sqlc.embed(p), sqlc.embed(a)
FROM apps a
INNER JOIN projects p ON p.id = a.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.slug = sqlc.arg(project_slug)
  AND a.slug = sqlc.arg(app_slug);
