-- name: UpdateGatewayReplicasAndHealth :exec
UPDATE gateways SET
replicas = sqlc.arg(replicas),
health = sqlc.arg(health),
updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
