-- name: DeleteOpenapiSpecsByDeploymentIds :exec
-- DeleteOpenapiSpecsByDeploymentIds removes generated specifications before
-- their deployments are hard-deleted by the retention sweep.
DELETE FROM openapi_specs
WHERE deployment_id IN (sqlc.slice('deployment_ids'));
