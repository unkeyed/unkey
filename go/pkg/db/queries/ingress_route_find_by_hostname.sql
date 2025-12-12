-- name: FindFrontlineRouteByHostname :one
SELECT * FROM frontline_routes WHERE hostname = ?;
