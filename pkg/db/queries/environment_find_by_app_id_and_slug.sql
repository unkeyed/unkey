-- name: FindEnvironmentByAppIdAndSlug :one
SELECT
  environments.id,
  environments.project_id
FROM environments
WHERE app_id = sqlc.arg(app_id) AND slug = sqlc.arg(slug);
