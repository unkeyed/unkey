-- name: FindAppByProjectAndSlug :one
SELECT sqlc.embed(apps)
FROM apps
WHERE apps.project_id = sqlc.arg(project_id)
  AND apps.slug = sqlc.arg(slug);
