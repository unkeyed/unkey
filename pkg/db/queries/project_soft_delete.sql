-- name: SoftDeleteProject :exec
UPDATE projects
SET deleted_at_m = sqlc.Arg(now)
WHERE id = sqlc.Arg(project_id);
