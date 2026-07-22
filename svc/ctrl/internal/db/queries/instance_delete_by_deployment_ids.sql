-- name: DeleteInstancesByDeploymentIds :exec
-- DeleteInstancesByDeploymentIds removes observed runtime state before its
-- deployments are hard-deleted by the retention sweep.
DELETE FROM instances
WHERE deployment_id IN (sqlc.slice('deployment_ids'));
