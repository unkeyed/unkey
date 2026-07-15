-- name: FindDeploymentWithEnvironmentAndApp :one
SELECT d.*, e.slug AS environment_slug, a.current_deployment_id, a.is_rolled_back
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.id = sqlc.arg(id);
