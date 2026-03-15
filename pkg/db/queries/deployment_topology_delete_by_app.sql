-- name: DeleteDeploymentTopologiesByAppId :exec
DELETE dt FROM deployment_topology dt
JOIN deployments d ON dt.deployment_id = d.id
WHERE d.app_id = sqlc.arg(app_id);
