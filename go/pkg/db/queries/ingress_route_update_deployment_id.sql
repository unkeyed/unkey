-- name: UpdateFrontlineRouteDeploymentId :exec
UPDATE frontline_routes
SET deployment_id = sqlc.arg(deploymentId)
WHERE id = sqlc.arg(id);
