-- name: FindDeploymentWithEnvironment :one
SELECT d.*, e.slug AS environment_slug
FROM deployments d
JOIN environments e ON e.id = d.environment_id
WHERE d.id = sqlc.arg(id);
