-- name: ProjectExists :one
SELECT EXISTS (
    SELECT 1
    FROM projects
    WHERE id = sqlc.arg(id)
    LIMIT 1
) AS project_exists;
