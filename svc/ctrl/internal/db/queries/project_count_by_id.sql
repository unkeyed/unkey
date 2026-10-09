-- name: CountProjectsById :one
-- CountProjectsById counts projects with an id. Only tests use this query.
SELECT COUNT(*)
FROM `projects`
WHERE id = sqlc.arg(id);
