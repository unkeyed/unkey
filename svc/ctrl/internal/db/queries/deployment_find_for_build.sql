-- name: FindDeploymentForBuild :one
SELECT id, workspace_id, project_id, app_id, environment_id, status, created_at,
       port, cpu_millicores, memory_mib, encrypted_environment_variables
FROM deployments
WHERE id = sqlc.arg(id);
