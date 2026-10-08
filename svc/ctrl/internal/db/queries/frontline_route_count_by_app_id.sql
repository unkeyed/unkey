-- name: CountFrontlineRoutesByAppId :one
-- CountFrontlineRoutesByAppId counts an app's frontline routes. Only tests use this query.
SELECT COUNT(*)
FROM `frontline_routes`
WHERE app_id = sqlc.arg(app_id);
