-- name: FindAppByProjectAndSlug :one
SELECT sqlc.embed(apps)
FROM apps
JOIN environments ON environments.id = apps.environment_id
WHERE apps.project_id = sqlc.arg(project_id)
  AND environments.slug = sqlc.arg(environment_slug)
  AND apps.slug = sqlc.arg(slug);
