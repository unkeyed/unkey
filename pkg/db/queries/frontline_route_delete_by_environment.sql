-- name: DeleteFrontlineRoutesByEnvironmentId :exec
DELETE FROM frontline_routes WHERE environment_id = sqlc.arg(environment_id);
