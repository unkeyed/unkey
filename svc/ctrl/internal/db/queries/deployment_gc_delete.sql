-- name: DeleteDeploymentByIDForGC :execrows
DELETE FROM deployments WHERE id = sqlc.arg(deployment_id);

-- name: SoftDeleteDeploymentForGC :execrows
UPDATE deployments SET deleted_at = sqlc.arg(deleted_at)
WHERE id = sqlc.arg(deployment_id) AND deleted_at IS NULL;

-- name: RestoreDeployment :execrows
UPDATE deployments SET deleted_at = NULL, restored_at = sqlc.arg(restored_at)
WHERE id = sqlc.arg(deployment_id)
  AND deleted_at > sqlc.arg(recovery_cutoff);

-- name: DeploymentExistsIncludingDeleted :one
SELECT EXISTS(SELECT 1 FROM deployments WHERE id = sqlc.arg(deployment_id));

-- name: DeploymentImageExistsIncludingDeleted :one
SELECT EXISTS(SELECT 1 FROM deployments WHERE image = sqlc.arg(image));

-- name: DeleteDeploymentStepsByDeploymentID :exec
DELETE FROM deployment_steps WHERE deployment_id = sqlc.arg(deployment_id);

-- name: DeleteDeploymentTopologiesByDeploymentID :exec
DELETE FROM deployment_topology WHERE deployment_id = sqlc.arg(deployment_id);

-- name: DeleteCiliumNetworkPoliciesByDeploymentID :exec
DELETE FROM cilium_network_policies WHERE deployment_id = sqlc.arg(deployment_id);

-- name: DeleteFrontlineRoutesByDeploymentID :exec
DELETE FROM frontline_routes WHERE deployment_id = sqlc.arg(deployment_id);

-- name: DeleteOpenAPISpecsByDeploymentID :exec
DELETE FROM openapi_specs WHERE deployment_id = sqlc.arg(deployment_id);

-- name: DeleteDeploymentChangesByDeploymentID :exec
DELETE FROM deployment_changes
WHERE resource_type = 'deployment_topology'
  AND resource_id = sqlc.arg(deployment_id);
