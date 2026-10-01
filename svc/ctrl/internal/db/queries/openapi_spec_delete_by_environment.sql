-- name: DeleteOpenApiSpecsByEnvironmentId :exec
DELETE oas FROM openapi_specs oas
JOIN deployments d ON d.id = oas.deployment_id
WHERE d.environment_id = sqlc.arg(environment_id);
