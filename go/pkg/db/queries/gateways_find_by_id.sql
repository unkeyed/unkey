-- name: FindGatewaysByID :many
SELECT * FROM gateways WHERE id = sqlc.arg(id);
