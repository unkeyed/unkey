-- name: DeleteDeploymentsByIds :exec
-- DeleteDeploymentsByIds removes deployments after cleanup has revalidated and
-- removed every dependent row in the same transaction.
DELETE FROM deployments
WHERE id IN (sqlc.slice('ids'));
