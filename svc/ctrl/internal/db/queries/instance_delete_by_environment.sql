-- name: DeleteInstancesByEnvironment :exec
DELETE i FROM instances i
JOIN deployments d ON d.id = i.deployment_id
WHERE d.environment_id = sqlc.arg(environment_id);
