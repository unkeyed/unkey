-- name: FindFrontlineRouteByID :one
SELECT *
FROM frontline_routes
WHERE id = ?;
