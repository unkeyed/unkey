-- name: FindAppByProjectAndSlug :one
SELECT
  apps.id,
  apps.workspace_id,
  apps.slug
FROM apps
WHERE apps.project_id = sqlc.arg(project_id)
  AND apps.slug = sqlc.arg(slug);
