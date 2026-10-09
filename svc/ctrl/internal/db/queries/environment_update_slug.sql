-- name: UpdateEnvironmentSlug :exec
UPDATE environments
SET slug = sqlc.arg(slug), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
