-- name: UpdateProjectDepotID :exec
UPDATE projects
SET
    depot_project_id = sqlc.arg(depot_project_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
