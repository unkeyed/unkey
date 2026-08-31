-- name: FindInstancesByDeploymentId :many
SELECT
 pk, id, deployment_id, workspace_id, project_id, app_id, region_id, k8s_name, address, cpu_millicores, memory_mib, storage_mib, status, container_status
FROM instances
WHERE deployment_id = sqlc.arg(deploymentId);
