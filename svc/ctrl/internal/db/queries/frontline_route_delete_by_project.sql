-- name: DeleteFrontlineRoutesByProjectId :exec
DELETE FROM frontline_routes WHERE project_id = sqlc.arg(project_id);
