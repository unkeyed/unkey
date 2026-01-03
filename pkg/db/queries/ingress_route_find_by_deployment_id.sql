-- name: FindFrontlineRoutesByDeploymentID :many
SELECT * FROM frontline_routes WHERE deployment_id = ?;
