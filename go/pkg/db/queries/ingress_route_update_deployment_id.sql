-- name: UpdateIngressRouteDeploymentId :exec
UPDATE ingress_routes
SET deployment_id = sqlc.arg(deploymentId)
WHERE id = sqlc.arg(id);
