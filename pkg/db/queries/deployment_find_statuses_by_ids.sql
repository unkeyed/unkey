-- name: FindDeploymentStatusesByIds :many
SELECT id, status
FROM deployments
WHERE id IN (sqlc.slice('ids'));
