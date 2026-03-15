-- name: DeleteFrontlineRoutesByAppId :exec
DELETE FROM frontline_routes WHERE app_id = sqlc.arg(app_id);
