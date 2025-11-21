-- name: FindDeploymentByID :one
SELECT * FROM `deployments` WHERE id = sqlc.arg(id);
