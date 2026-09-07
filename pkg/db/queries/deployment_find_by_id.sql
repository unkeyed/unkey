-- name: FindDeploymentById :one
SELECT * FROM `deployments` WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
