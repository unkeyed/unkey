-- name: DeleteDeploymentStepsByAppId :exec
DELETE ds FROM deployment_steps ds
JOIN deployments d ON ds.deployment_id = d.id
WHERE d.app_id = sqlc.arg(app_id);
