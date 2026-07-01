-- name: FindEnvironmentByAppIdAndSlug :one
SELECT sqlc.embed(environments) FROM environments
WHERE app_id = sqlc.arg(app_id) AND slug = sqlc.arg(slug);
