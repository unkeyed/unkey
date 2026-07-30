-- name: DeleteDeploymentStepsByEnvironmentId :exec
DELETE ds FROM deployment_steps ds
JOIN deployments d ON d.id = ds.deployment_id
WHERE d.environment_id = sqlc.arg(environment_id);
