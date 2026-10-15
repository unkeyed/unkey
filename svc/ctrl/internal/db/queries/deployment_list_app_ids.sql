-- name: ListDeploymentAppIds :many
SELECT id, app_id FROM deployments WHERE id IN (sqlc.slice(ids));
