-- name: ListProjectDepotReferences :many
SELECT pk, id, depot_project_id
FROM projects
WHERE pk > sqlc.arg(pagination_cursor)
  AND depot_project_id IS NOT NULL
ORDER BY pk
LIMIT ?;

-- name: ProjectDepotIDExists :one
SELECT EXISTS(
  SELECT 1 FROM projects WHERE depot_project_id = sqlc.arg(depot_project_id)
) AS referenced;

-- name: ClearProjectDepotIDIfMatches :execrows
UPDATE projects
SET depot_project_id = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(project_id)
  AND depot_project_id = sqlc.arg(depot_project_id);
