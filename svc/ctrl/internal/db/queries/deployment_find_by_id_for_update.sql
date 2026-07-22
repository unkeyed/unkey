-- name: FindDeploymentByIdForUpdate :one
-- FindDeploymentByIdForUpdate locks a routing target so retention cleanup
-- cannot delete it while routes and the app live pointer are reassigned in the
-- same transaction.
SELECT * FROM deployments
WHERE id = sqlc.arg(id)
FOR UPDATE;
