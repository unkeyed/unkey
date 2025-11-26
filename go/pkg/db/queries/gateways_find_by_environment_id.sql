-- name: FindGatewaysByEnvironmentID :many
SELECT * FROM gateways WHERE environment_id = sqlc.arg(environment_id);
