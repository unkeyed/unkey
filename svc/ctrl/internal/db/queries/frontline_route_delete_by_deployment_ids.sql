-- name: DeleteFrontlineRoutesByDeploymentIds :exec
-- DeleteFrontlineRoutesByDeploymentIds removes routes that can no longer serve
-- before their deployments are hard-deleted by the retention sweep.
DELETE FROM frontline_routes
WHERE deployment_id IN (sqlc.slice('deployment_ids'));
