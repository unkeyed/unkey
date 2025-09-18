-- name: UpsertGateway :exec
INSERT INTO gateways (
workspace_id,
deployment_id,
hostname,
config
)
VALUES (
sqlc.arg(workspace_id),
sqlc.arg(deployment_id),
sqlc.arg(hostname),
sqlc.arg(config)
)
ON DUPLICATE KEY UPDATE
    workspace_id = sqlc.arg(workspace_id),
    deployment_id = sqlc.arg(deployment_id),
    hostname = sqlc.arg(hostname),
    config = sqlc.arg(config);
