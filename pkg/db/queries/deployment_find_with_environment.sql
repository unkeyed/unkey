-- name: FindDeploymentWithEnvironment :one
SELECT d.*, e.slug AS environment_slug, e.kind AS environment_kind
FROM deployments d
JOIN environments e ON d.environment_id = e.id
WHERE d.id = sqlc.arg(id);
