-- name: FindDeploymentWithEnvironment :one
SELECT d.*, e.slug AS environment_slug, e.is_production AS environment_is_production
FROM deployments d
JOIN environments e ON d.environment_id = e.id
WHERE d.id = sqlc.arg(id);
