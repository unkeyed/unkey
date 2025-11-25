-- name: FindIngressRouteByHostname :one
SELECT * FROM ingress_routes WHERE hostname = ?;
