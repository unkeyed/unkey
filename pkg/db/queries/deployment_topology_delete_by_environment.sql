-- name: DeleteDeploymentTopologiesByEnvironmentId :exec
DELETE dt FROM deployment_topology dt
JOIN deployments d ON dt.deployment_id = d.id
WHERE d.environment_id = sqlc.arg(environment_id);
