-- name: FindGatewayByID :one
SELECT * FROM gateways WHERE id = sqlc.arg(id) LIMIT 1;
