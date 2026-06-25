-- name: DeleteDeploymentStepsByEnvironmentId :exec
DELETE ds FROM deployment_steps ds
JOIN deployments d ON ds.deployment_id = d.id
WHERE d.environment_id = sqlc.arg(environment_id);
