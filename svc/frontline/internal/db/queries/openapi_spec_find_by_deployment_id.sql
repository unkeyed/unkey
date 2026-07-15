-- name: FindOpenApiSpecByDeploymentID :one
-- FindOpenApiSpecByDeploymentID returns the scraped OpenAPI spec for a
-- deployment. Frontline uses this to hydrate openapi policies that don't
-- carry an inline spec.
SELECT content FROM openapi_specs WHERE deployment_id = sqlc.arg(deployment_id);
