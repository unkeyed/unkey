-- name: ListDeploymentAppIds :many
SELECT DISTINCT app_id FROM deployments WHERE id IN (sqlc.slice(ids));
