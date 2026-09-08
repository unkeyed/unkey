-- name: FindAppByWorkspaceAndSlugs :one
SELECT
  p.id AS project_id,
  a.id AS app_id
FROM apps a
INNER JOIN projects p ON a.project_id = p.id
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.slug = sqlc.arg(project_slug)
  AND a.slug = sqlc.arg(app_slug);
