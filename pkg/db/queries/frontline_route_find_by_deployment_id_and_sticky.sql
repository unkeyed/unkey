-- name: FindFrontlineRouteByDeploymentIDAndSticky :one
SELECT * FROM frontline_routes WHERE deployment_id = ? AND sticky = ?;
