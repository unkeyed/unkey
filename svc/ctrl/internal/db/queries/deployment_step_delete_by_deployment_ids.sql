-- name: DeleteDeploymentStepsByDeploymentIds :exec
-- DeleteDeploymentStepsByDeploymentIds removes execution history before its
-- deployments are hard-deleted by the retention sweep.
DELETE FROM deployment_steps
WHERE deployment_id IN (sqlc.slice('deployment_ids'));
