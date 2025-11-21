-- name: InsertGateway :exec
INSERT INTO gateways (
id,
workspace_id,
k8s_service_name,
region,
image,
health,
replicas

)
VALUES (
sqlc.arg(id),
sqlc.arg(workspace_id),
sqlc.arg(k8s_service_name),
sqlc.arg(region),
sqlc.arg(image),
sqlc.arg(health),
sqlc.arg(replicas)

)
