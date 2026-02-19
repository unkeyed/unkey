-- name: FindAppByProjectAndSlug :one
SELECT sqlc.embed(apps)
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND slug = sqlc.arg(slug);
