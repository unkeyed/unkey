-- name: FindIngressRoutesByDeploymentID :many
SELECT * FROM ingress_routes WHERE deployment_id = ?;
